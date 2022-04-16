package minecraft

import (
	"context"
	"fmt"
	"log"

	"github.com/Adirelle/mcvisor/pkg/events"
	properties "github.com/dmotylev/goproperties"
)

type (
	RemoteConsoleService struct {
		propertyPath string
		events.Handler
		*rconSettings
	}

	rconSettings struct {
		Enabled  bool
		Port     uint16
		Password string
	}
)

func MakeRemoteConsoleService(conf Config, handler events.Handler) RemoteConsoleService {
	return RemoteConsoleService{
		propertyPath: conf.ServerPropertiesPath(),
		Handler:      handler,
		rconSettings: new(rconSettings),
	}
}

func (r RemoteConsoleService) Serve(ctx context.Context) error {
	if err := r.readSettings(); err != nil {
		return err
	}
	log.Printf("rcon settings: %#v", r.rconSettings)

	<-ctx.Done()

	return nil
}

func (r RemoteConsoleService) readSettings() error {
	props, err := properties.Load(r.propertyPath)
	if err != nil {
		return fmt.Errorf("could not read server properties `%s`: %w", r.propertyPath, err)
	}

	r.Enabled = props.Bool("enable-rcon", false)
	r.Port = uint16(props.Int("rcon.port", 25575))
	r.Password = props.String("rcon.password", "")
	return nil
}
