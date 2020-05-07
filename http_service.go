package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	errTimestampExpired = errors.New("securecookie: expired timestamp")
)

// HTTPService provides an http service.
type HTTPService struct {
	HTTPConfig
	ucs *UserCounterService
	*http.Server
	errch  chan error
	router *httprouter.Router
	sigch  chan os.Signal
}

// NewHTTPService creates a new http service.
func NewHTTPService(config HTTPConfig) (*HTTPService, error) {
	var store = sessions.NewCookieStore([]byte(os.Getenv("T_SESSION_KEY")))
	store.MaxAge(300)

	s := &HTTPService{
		Server:     &http.Server{},
		HTTPConfig: config,
		errch:      make(chan error, 1),
		sigch:      make(chan os.Signal, 1),
		ucs: &UserCounterService{
			mu:            sync.RWMutex{},
			store:         store,
			CacheInterval: config.CacheInterval,
		},
	}

	s.router = httprouter.New()
	s.Handler = s.router

	return s, nil
}

// Run runs the http service.
func (s *HTTPService) Run(ctx context.Context) error {
	signal.Notify(s.sigch, syscall.SIGINT, syscall.SIGTERM)

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.HTTPConfig.Host, s.HTTPConfig.Port))
	if err != nil {
		return errors.Wrap(err, "creating tcp listener")
	}

	go func() {
		logrus.
			WithField("addr", l.Addr()).
			Info("http service running...")

		if err := s.Serve(l); err != nil {
			s.errch <- err
		}

		close(s.errch)
	}()

	select {
	case <-s.sigch:
	case <-ctx.Done():
	case err := <-s.errch:
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, s.HTTPConfig.ShutdownTimeout)
	defer cancel()
	return s.Shutdown(ctx)
}

// UserCounterService
type UserCounterService struct {
	mu            sync.RWMutex
	onlineUsers   int
	store         *sessions.CookieStore
	CacheInterval time.Duration
}

func (ucs *UserCounterService) addToStore(w http.ResponseWriter, r *http.Request, session *sessions.Session) {
	session.Values["__id"] = uuid.New().String()
	session.Values["__addr"] = r.RemoteAddr

	if err := session.Save(r, w); err != nil {
		logrus.Error(err, "cannot save session")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ucs.mu.Lock()
	ucs.onlineUsers += 1
	ucs.mu.Unlock()
}

func (ucs *UserCounterService) updateStore(w http.ResponseWriter, r *http.Request) {
	session := sessions.NewSession(ucs.store, "__token")
	session.Options = ucs.store.Options

	ucs.addToStore(w, r, session)
}

func (s *HTTPService) initRoutes() error {
	s.router.GET("/users", s.countUsers)

	return nil
}

// countUsers handler has params argument b'case it's needed by httprouter signature
func (s *HTTPService) countUsers(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	logrus.Info(r.RemoteAddr)

	session, err := s.ucs.store.Get(r, "__token")
	if err != nil {
		if err.Error() == errTimestampExpired.Error() {
			s.ucs.updateStore(w, r)
			return
		}
		logrus.Error(err, ": cannot create or get session")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if session.IsNew {
		s.ucs.addToStore(w, r, session)
	}


	w.Write([]byte(fmt.Sprintf("online users: %v", s.ucs.onlineUsers)))
}
