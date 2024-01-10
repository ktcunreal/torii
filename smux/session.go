package smux

import (
	"container/heap"
	"errors"
	"io"
	"net"
	"fmt"
	"sync"
	"golang.org/x/crypto/nacl/secretbox"
	"sync/atomic"
	"time"
)

const (
	defaultAcceptBacklog = 1024
	maxShaperSize        = 1024
	openCloseTimeout     = 30 * time.Second // stream open/close timeout
)

// define frame class
type CLASSID int

const (
	CLSCTRL CLASSID = iota
	CLSDATA
)

var (
	ErrInvalidProtocol = errors.New("invalid protocol")
	ErrConsumed        = errors.New("peer consumed more than sent")
	ErrGoAway          = errors.New("stream id overflows, should start a new connection")
	ErrTimeout         = errors.New("timeout")
	ErrWouldBlock      = errors.New("operation would block on IO")
)

type writeRequest struct {
	class  CLASSID
	frame  Frame
	seq    uint32
	result chan writeResult
}

type writeResult struct {
	n   int
	err error
}

// 


// Session defines a multiplexed connection for streams
type Session struct {
	conn io.ReadWriteCloser
	
	config           *Config
	nextStreamID     uint32 // next stream identifier
	nextStreamIDLock sync.Mutex

	bucket       int32         // token bucket
	bucketNotify chan struct{} // used for waiting for tokens

	streams    map[uint32]*Stream // all streams in this session
	streamLock sync.Mutex         // locks streams

	die     chan struct{} // flag session has died
	dieOnce sync.Once

	// socket error handling
	socketReadError      atomic.Value
	socketWriteError     atomic.Value
	chSocketReadError    chan struct{}
	chSocketWriteError   chan struct{}
	socketReadErrorOnce  sync.Once
	socketWriteErrorOnce sync.Once

	// smux protocol errors
	protoError     atomic.Value
	chProtoError   chan struct{}
	protoErrorOnce sync.Once

	chAccepts chan *Stream

	dataReady int32 // flag data has arrived

	goAway int32 // flag id exhausted

	deadline atomic.Value

	requestID uint32            // write request monotonic increasing
	shaper    chan writeRequest // a shaper for writing
	writes    chan writeRequest

	isClient	bool
	ServerSN, ServerRN, ClientSN, ClientRN [24]byte
	NaclKey [32]byte
	keyring  *Keyring
}

func newSession(config *Config, conn io.ReadWriteCloser, client bool) *Session {
	s := new(Session)
	s.die = make(chan struct{})
	s.conn = conn
	s.config = config
	s.streams = make(map[uint32]*Stream)
	s.chAccepts = make(chan *Stream, defaultAcceptBacklog)
	s.bucket = int32(config.MaxReceiveBuffer)
	s.bucketNotify = make(chan struct{}, 1)
	s.shaper = make(chan writeRequest)
	s.writes = make(chan writeRequest)
	s.chSocketReadError = make(chan struct{})
	s.chSocketWriteError = make(chan struct{})
	s.chProtoError = make(chan struct{})
	s.keyring = NewKeyring("abc")

	if client {
		s.nextStreamID = 1
		s.isClient = true
	} else {
		s.nextStreamID = 0
		s.isClient = false
	}

	copy(s.NaclKey[:], s.keyring.Extract(nil, "nacl"))

	go s.shaperLoop()
	go s.recvLoop()
	go s.sendLoop()
	if !config.KeepAliveDisabled {
		go s.keepalive()
	}
	return s
}

// OpenStream is used to create a new stream
func (s *Session) OpenStream() (*Stream, error) {
	if s.IsClosed() {
		return nil, io.ErrClosedPipe
	}

	// generate stream id
	s.nextStreamIDLock.Lock()
	if s.goAway > 0 {
		s.nextStreamIDLock.Unlock()
		return nil, ErrGoAway
	}

	s.nextStreamID += 2
	sid := s.nextStreamID
	if sid == sid%2 { // stream-id overflows
		s.goAway = 1
		s.nextStreamIDLock.Unlock()
		return nil, ErrGoAway
	}
	s.nextStreamIDLock.Unlock()

	stream := newStream(sid, s.config.MaxFrameSize, s)

	if _, err := s.writeFrame(newFrame(byte(s.config.Version), cmdSYN, sid)); err != nil {
		return nil, err
	}

	s.streamLock.Lock()
	defer s.streamLock.Unlock()
	select {
	case <-s.chSocketReadError:
		return nil, s.socketReadError.Load().(error)
	case <-s.chSocketWriteError:
		return nil, s.socketWriteError.Load().(error)
	case <-s.die:
		return nil, io.ErrClosedPipe
	default:
		s.streams[sid] = stream
		return stream, nil
	}
}

// Open returns a generic ReadWriteCloser
func (s *Session) Open() (io.ReadWriteCloser, error) {
	return s.OpenStream()
}

// AcceptStream is used to block until the next available stream
// is ready to be accepted.
func (s *Session) AcceptStream() (*Stream, error) {
	var deadline <-chan time.Time
	if d, ok := s.deadline.Load().(time.Time); ok && !d.IsZero() {
		timer := time.NewTimer(time.Until(d))
		defer timer.Stop()
		deadline = timer.C
	}

	select {
	case stream := <-s.chAccepts:
		return stream, nil
	case <-deadline:
		return nil, ErrTimeout
	case <-s.chSocketReadError:
		return nil, s.socketReadError.Load().(error)
	case <-s.chProtoError:
		return nil, s.protoError.Load().(error)
	case <-s.die:
		return nil, io.ErrClosedPipe
	}
}

// Accept Returns a generic ReadWriteCloser instead of smux.Stream
func (s *Session) Accept() (io.ReadWriteCloser, error) {
	return s.AcceptStream()
}

// Close is used to close the session and all streams.
func (s *Session) Close() error {
	var once bool
	s.dieOnce.Do(func() {
		close(s.die)
		once = true
	})

	if once {
		s.streamLock.Lock()
		for k := range s.streams {
			s.streams[k].sessionClose()
		}
		s.streamLock.Unlock()
		return s.conn.Close()
	} else {
		return io.ErrClosedPipe
	}
}

// CloseChan can be used by someone who wants to be notified immediately when this
// session is closed
func (s *Session) CloseChan() <-chan struct{} {
	return s.die
}

// notifyBucket notifies recvLoop that bucket is available
func (s *Session) notifyBucket() {
	select {
	case s.bucketNotify <- struct{}{}:
	default:
	}
}

func (s *Session) notifyReadError(err error) {
	s.socketReadErrorOnce.Do(func() {
		s.socketReadError.Store(err)
		close(s.chSocketReadError)
	})
}

func (s *Session) notifyWriteError(err error) {
	s.socketWriteErrorOnce.Do(func() {
		s.socketWriteError.Store(err)
		close(s.chSocketWriteError)
	})
}

func (s *Session) notifyProtoError(err error) {
	s.protoErrorOnce.Do(func() {
		s.protoError.Store(err)
		close(s.chProtoError)
	})
}

// IsClosed does a safe check to see if we have shutdown
func (s *Session) IsClosed() bool {
	select {
	case <-s.die:
		return true
	default:
		return false
	}
}

// NumStreams returns the number of currently open streams
func (s *Session) NumStreams() int {
	if s.IsClosed() {
		return 0
	}
	s.streamLock.Lock()
	defer s.streamLock.Unlock()
	return len(s.streams)
}

// SetDeadline sets a deadline used by Accept* calls.
// A zero time value disables the deadline.
func (s *Session) SetDeadline(t time.Time) error {
	s.deadline.Store(t)
	return nil
}

// LocalAddr satisfies net.Conn interface
func (s *Session) LocalAddr() net.Addr {
	if ts, ok := s.conn.(interface {
		LocalAddr() net.Addr
	}); ok {
		return ts.LocalAddr()
	}
	return nil
}

// RemoteAddr satisfies net.Conn interface
func (s *Session) RemoteAddr() net.Addr {
	if ts, ok := s.conn.(interface {
		RemoteAddr() net.Addr
	}); ok {
		return ts.RemoteAddr()
	}
	return nil
}

// notify the session that a stream has closed
func (s *Session) streamClosed(sid uint32) {
	s.streamLock.Lock()
	if n := s.streams[sid].recycleTokens(); n > 0 { // return remaining tokens to the bucket
		if atomic.AddInt32(&s.bucket, int32(n)) > 0 {
			s.notifyBucket()
		}
	}
	delete(s.streams, sid)
	s.streamLock.Unlock()
}

// returnTokens is called by stream to return token after read
func (s *Session) returnTokens(n int) {
	if atomic.AddInt32(&s.bucket, int32(n)) > 0 {
		s.notifyBucket()
	}
}

// recvLoop keeps on reading from underlying connection if tokens are available
func (s *Session) recvLoop() {
	//var hdr rawHeader
	var updHdr updHeader
	// var ehdr encryptedHeader
	ehdr := NewEncryptedHeader(s.keyring)
	
	for {
		for atomic.LoadInt32(&s.bucket) <= 0 && !s.IsClosed() {
			select {
			case <-s.bucketNotify:
			case <-s.die:
				return
			}
		}

		// read header first
		if _, err := io.ReadFull(s.conn, ehdr.eb[:]); err == nil {
			atomic.StoreInt32(&s.dataReady, 1)
			ehdr.Unmask()
			// Get sid
			sid := ehdr.StreamID()

			switch ehdr.CMD() {
			case cmdNOP:
			case cmdSYN:
				s.streamLock.Lock()
				if _, ok := s.streams[sid]; !ok {
					stream := newStream(sid, s.config.MaxFrameSize, s)
					s.streams[sid] = stream
					select {
					case s.chAccepts <- stream:
					case <-s.die:
					}
				}
				s.streamLock.Unlock()
			case cmdFIN:
				s.streamLock.Lock()
				if stream, ok := s.streams[sid]; ok {
					stream.fin()
					stream.notifyReadEvent()
				}
				s.streamLock.Unlock()
			case cmdPSH:
				if ehdr.Length() > 0 {	
					ebuf := defaultAllocator.Get(int(ehdr.Length()))
					if written, err := io.ReadFull(s.conn, ebuf); err == nil {
						var plain []byte
						var ok bool

						if s.isClient {
							plain, ok = secretbox.Open(nil, ebuf[:], &s.ClientRN, &s.NaclKey)
							if !ok {
								fmt.Println("Decrypt Failed. Send conn to jail")
								return
							} 
							increment(&s.ClientRN)
						} else {
							plain, ok = secretbox.Open(nil, ebuf[:], &s.ServerRN, &s.NaclKey)
							if !ok {
								fmt.Println("Decrypt Failed")
								return
							}
							increment(&s.ServerRN)	
						}

						s.streamLock.Lock()
						if stream, ok := s.streams[sid]; ok {
							stream.pushBytes(plain)
							atomic.AddInt32(&s.bucket, -int32(written))
							stream.notifyReadEvent()
						}
						s.streamLock.Unlock()
						
					} else {
						s.notifyReadError(err)
						return
					}
				}
			case cmdUPD:
				if _, err := io.ReadFull(s.conn, updHdr[:]); err == nil {
					s.streamLock.Lock()
					if stream, ok := s.streams[sid]; ok {
						stream.update(updHdr.Consumed(), updHdr.Window())
					}
					s.streamLock.Unlock()
				} else {
					s.notifyReadError(err)
					return
				}
			default:
				s.notifyProtoError(ErrInvalidProtocol)
				return
			}
		} else {
			s.notifyReadError(err)
			return
		}
	}
}

func (s *Session) keepalive() {
	tickerPing := time.NewTicker(s.config.KeepAliveInterval)
	tickerTimeout := time.NewTicker(s.config.KeepAliveTimeout)
	defer tickerPing.Stop()
	defer tickerTimeout.Stop()
	for {
		select {
		case <-tickerPing.C:
			s.writeFrameInternal(newFrame(byte(s.config.Version), cmdNOP, 0), tickerPing.C, CLSCTRL)
			s.notifyBucket() // force a signal to the recvLoop
		case <-tickerTimeout.C:
			if !atomic.CompareAndSwapInt32(&s.dataReady, 1, 0) {
				// recvLoop may block while bucket is 0, in this case,
				// session should not be closed.
				if atomic.LoadInt32(&s.bucket) > 0 {
					s.Close()
					return
				}
			}
		case <-s.die:
			return
		}
	}
}

// shaper shapes the sending sequence among streams
func (s *Session) shaperLoop() {
	var reqs shaperHeap
	var next writeRequest
	var chWrite chan writeRequest
	var chShaper chan writeRequest

	for {
		// chWrite is not available until it has packet to send
		if len(reqs) > 0 {
			chWrite = s.writes
			next = heap.Pop(&reqs).(writeRequest)
		} else {
			chWrite = nil
		}

		// control heap size, chShaper is not available until packets are less than maximum allowed
		if len(reqs) >= maxShaperSize {
			chShaper = nil
		} else {
			chShaper = s.shaper
		}

		// assertion on non nil
		if chShaper == nil && chWrite == nil {
			panic("both channel are nil")
		}

		select {
		case <-s.die:
			return
		case r := <-chShaper:
			if chWrite != nil { // next is valid, reshape
				heap.Push(&reqs, next)
			}
			heap.Push(&reqs, r)
		case chWrite <- next:
		}
	}
}

func (s *Session) sendLoop() {
	var n int
	var err error
	ehdr := NewEncryptedHeader(s.keyring)
	for {
		select {
		case <-s.die:
			return
		case request := <-s.writes:
			// Process payload by cmd
			if request.frame.cmd == 2 {
				var cipher []byte
				
				// Encrypt data block
				if s.isClient {
					cipher = secretbox.Seal([]byte{}, request.frame.data, &s.ClientSN, &s.NaclKey)
					increment(&s.ClientSN)
				} else {
					cipher = secretbox.Seal([]byte{}, request.frame.data, &s.ServerSN, &s.NaclKey)
					increment(&s.ServerSN)
				}

				// Set Length
				ehdr.SetEncryptedHeader(request.frame.cmd, request.frame.sid, uint16(len(cipher)))
				ehdr.Mask()

				// Make send buffer and copy cipher to buffer
				buf := make([]byte, encryptedHeaderSize + len(cipher))

				copy(buf[:encryptedHeaderSize], ehdr.eb[:])
				copy(buf[encryptedHeaderSize:], cipher)
				
				// Write via conn
				_, err = s.conn.Write(buf[:encryptedHeaderSize+len(cipher)])

				// Set wrote bytes
				n = len(request.frame.data)
				if err != nil {
					n = 0
				}
			} else {
				ehdr.SetEncryptedHeader(request.frame.cmd, request.frame.sid, uint16(len(request.frame.data)))
				ehdr.Mask()

				// Make send buffer and copy raw data to buffer
				buf := make([]byte, encryptedHeaderSize + len(request.frame.data))
				
				copy(buf[:encryptedHeaderSize], ehdr.eb[:])
				copy(buf[encryptedHeaderSize:], request.frame.data)

				// Write via conn
				n, err = s.conn.Write(ehdr.eb[:encryptedHeaderSize+len(request.frame.data)])
	
				// Set wrote bytes, subtract raw header size from n
				n -= headerSize
				if n < 0 {
					n = 0
				}
			}
			
			result := writeResult{
				n:   n,
				err: err,
			}

			request.result <- result
			close(request.result)

			// store conn error
			if err != nil {
				s.notifyWriteError(err)
				return
			}
		}
	}
}

// writeFrame writes the frame to the underlying connection
// and returns the number of bytes written if successful
func (s *Session) writeFrame(f Frame) (n int, err error) {
	return s.writeFrameInternal(f, time.After(openCloseTimeout), CLSCTRL)
}


// internal writeFrame version to support deadline used in keepalive
func (s *Session) writeFrameInternal(f Frame, deadline <-chan time.Time, class CLASSID) (int, error) {
	req := writeRequest{
		class:  class,
		frame:  f,
		seq:    atomic.AddUint32(&s.requestID, 1),
		result: make(chan writeResult, 1),
	}
	select {
	case s.shaper <- req:
	case <-s.die:
		return 0, io.ErrClosedPipe
	case <-s.chSocketWriteError:
		return 0, s.socketWriteError.Load().(error)
	case <-deadline:
		return 0, ErrTimeout
	}

	select {
	case result := <-req.result:
		return result.n, result.err
	case <-s.die:
		return 0, io.ErrClosedPipe
	case <-s.chSocketWriteError:
		return 0, s.socketWriteError.Load().(error)
	case <-deadline:
		return 0, ErrTimeout
	}
}
