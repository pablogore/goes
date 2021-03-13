package handler_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/modernice/goes/command"
	"github.com/modernice/goes/command/cmdbus"
	"github.com/modernice/goes/command/encoding"
	"github.com/modernice/goes/command/handler"
	mock_command "github.com/modernice/goes/command/mocks"
	"github.com/modernice/goes/event/eventbus/chanbus"
)

type recorder struct {
	h      func(context.Context, command.Command) error
	params chan handlerParams
}

type handlerParams struct {
	ctx context.Context
	cmd command.Command
}

type mockPayload struct{}

func TestHandler_On(t *testing.T) {
	enc := encoding.NewGobEncoder()
	enc.Register("foo", func() command.Payload {
		return mockPayload{}
	})
	ebus := chanbus.New()
	bus := cmdbus.New(enc, ebus)
	h := handler.New(bus)

	rec := newRecorder(nil)
	errs, err := h.On(context.Background(), "foo", rec.Handle)
	if err != nil {
		t.Fatalf("On shouldn't fail; failed with %q", err)
	}

	cmd := command.New("foo", mockPayload{})
	if err := bus.Dispatch(context.Background(), cmd); err != nil {
		t.Fatalf("failed to dispatch Command: %v", err)
	}

	timeout, stop := after(3 * time.Second)
	defer stop()

	select {
	case <-timeout:
		t.Fatalf("didn't receive Command after %s", 3*time.Second)
	case err, ok := <-errs:
		if ok {
			t.Fatal(err)
		}
	case p := <-rec.params:
		if p.ctx == nil {
			t.Errorf("handler received <nil> Context!")
		}

		if !reflect.DeepEqual(p.cmd, cmd) {
			t.Errorf("handler received wrong Command. want=%v got=%v", cmd, p.cmd)
		}
	}
}

func TestHandler_On_cancelContext(t *testing.T) {
	enc := encoding.NewGobEncoder()
	enc.Register("foo", func() command.Payload {
		return mockPayload{}
	})
	ebus := chanbus.New()
	bus := cmdbus.New(enc, ebus, cmdbus.AssignTimeout(500*time.Millisecond))
	h := handler.New(bus)

	rec := newRecorder(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs, err := h.On(ctx, "foo", rec.Handle)
	if err != nil {
		t.Fatalf("On shouldn't fail; failed with %q", err)
	}
	cancel()

	timeout, stop := after(10 * time.Millisecond)
	defer stop()
	<-timeout

	cmd := command.New("foo", mockPayload{})
	if err := bus.Dispatch(context.Background(), cmd); err == nil {
		t.Fatal("dispatch should have failed, but didn't!")
	}

	timeout, stop = after(10 * time.Millisecond)
	defer stop()

	select {
	case <-rec.params:
		t.Fatalf("handler should not have received a Command!")
	case err, ok := <-errs:
		if ok {
			t.Fatal(err)
		}
	case <-timeout:
	}
}

func TestHandler_On_subscribeError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bus := mock_command.NewMockBus(ctrl)
	h := handler.New(bus)

	mockError := errors.New("mock error")
	bus.EXPECT().Subscribe(gomock.Any(), "foo").Return(nil, nil, mockError)

	rec := newRecorder(nil)

	_, err := h.On(context.Background(), "foo", rec.Handle)
	if !errors.Is(err, mockError) {
		t.Errorf("On should fail with %q; got %q", mockError, err)
	}
}

func TestHandler_busError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	bus := mock_command.NewMockBus(ctrl)
	h := handler.New(bus)

	mockCmds := make(chan command.Context)
	mockErrs := make(chan error)
	bus.EXPECT().Subscribe(gomock.Any(), "foo").Return(mockCmds, mockErrs, nil)

	errs, err := h.On(context.Background(), "foo", func(ctx context.Context, cmd command.Command) error {
		return nil
	})
	if err != nil {
		t.Fatalf("On shouldn't fail; failed with %q", err)
	}

	mockError := errors.New("mock error")
	mockErrs <- mockError

	timeout, stop := after(200 * time.Millisecond)
	defer stop()

	select {
	case <-timeout:
		t.Fatalf("didn't receive error after %s", 200*time.Millisecond)
	case err := <-errs:
		if err != mockError {
			t.Errorf("received wrong error. want=%q got=%q", mockError, err)
		}
	}
}

func newRecorder(h func(context.Context, command.Command) error) *recorder {
	if h == nil {
		h = func(context.Context, command.Command) error { return nil }
	}
	return &recorder{
		h:      h,
		params: make(chan handlerParams, 1),
	}
}

func (r *recorder) Handle(ctx context.Context, cmd command.Command) error {
	r.params <- handlerParams{ctx, cmd}
	return r.h(ctx, cmd)
}

func after(d time.Duration) (<-chan time.Time, func()) {
	timer := time.NewTimer(d)
	return timer.C, func() {
		timer.Stop()
	}
}
