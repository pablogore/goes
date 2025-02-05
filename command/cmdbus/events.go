package cmdbus

import (
	"time"

	"github.com/google/uuid"
	"github.com/modernice/goes/codec"
)

const (
	// CommandDispatched is published by a Bus to dispatch a Command.
	CommandDispatched = "goes.command.dispatched"

	// CommandRequested is published by a Bus to show interest in a dispatched
	// Command.
	CommandRequested = "goes.command.requested"

	// CommandAssigned is published by a Bus to assign a dispatched Command to a
	// Handler.
	CommandAssigned = "goes.command.assigned"

	// CommandAccepted is published by a Bus to notify other Buses that a
	// Command has been accepted.
	CommandAccepted = "goes.command.accepted"

	// CommandExecuted is published by a Bus to notify other Buses that a
	// Command has been executed.
	CommandExecuted = "goes.command.executed"
)

// CommandDispatchedData is the event Data for the CommandDispatched Event.
type CommandDispatchedData struct {
	// ID is the unique Command ID.
	ID uuid.UUID

	// Name is the name of the Command.
	Name string

	// AggregateName is the name of the  aggregate the Command belongs to.
	// (optional)
	AggregateName string

	// AggregateID is the ID of the aggregate the Command belongs to. (optional)
	AggregateID uuid.UUID

	// Payload is the encoded domain-specific Command Payload.
	Payload []byte
}

// CommandRequestedData is the event Data for the CommandRequested Event.
type CommandRequestedData struct {
	ID    uuid.UUID
	BusID uuid.UUID
}

// CommandAssignedData is the event Data for the CommandAssigned Event.
type CommandAssignedData struct {
	ID    uuid.UUID
	BusID uuid.UUID
}

// CommandAcceptedData is the event Data for the CommandAccepted Event.
type CommandAcceptedData struct {
	ID    uuid.UUID
	BusID uuid.UUID
}

// CommandExecutedData is the event Data for the CommandExecuted Event.
type CommandExecutedData struct {
	ID      uuid.UUID
	Runtime time.Duration
	Error   string
}

// RegisterEvents registers the command events into a Registry.
func RegisterEvents(reg *codec.Registry) {
	gob := codec.Gob(reg)
	gob.GobRegister(CommandDispatched, func() any {
		return CommandDispatchedData{}
	})
	gob.GobRegister(CommandRequested, func() any {
		return CommandRequestedData{}
	})
	gob.GobRegister(CommandAssigned, func() any {
		return CommandAssignedData{}
	})
	gob.GobRegister(CommandAccepted, func() any {
		return CommandAcceptedData{}
	})
	gob.GobRegister(CommandExecuted, func() any {
		return CommandExecutedData{}
	})
}
