package gorabbit

import (
	"github.com/dimonrus/gocli"
	"github.com/dimonrus/porterr"
)

const (
	CommandStart = "start"
	CommandStop = "stop"
	CommandRestart = "restart"
	CommandStatus = "status"
	CommandConsumer = "consumer"
	CommandSet = "set"

	CommandKeyWordAll = "all"
	CommandKeyWordCount = "count"
)

// Parse gocli.Command
func ParseCommand(command gocli.Command) (action string, arguments []gocli.Argument, e porterr.IError)  {
	args := command.Arguments()
	if len(args) > 0 {
		if args[0].Name != CommandConsumer {
			e = porterr.New(porterr.PortErrorArgument, "Consumer command must start with keyword 'consumer'")
			return
		}
		if len(args) < 3 {
			e = porterr.New(porterr.PortErrorArgument, "Consumer command must contain action with argument")
			return
		}
		action = args[1].Name
		arguments = args[2:]
	} else {
		e = porterr.New(porterr.PortErrorArgument, "Consumer command is empty")
	}
	return
}
