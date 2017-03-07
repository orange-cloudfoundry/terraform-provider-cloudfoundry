package cf_client

import (
	"code.cloudfoundry.org/cli/cf/trace"
	"log"
)

type CfLogger struct {
	writesToConsole bool
}

func NewCfLogger(writesToConsole bool) trace.Printer {
	return &CfLogger{
		writesToConsole: writesToConsole,
	}
}

func (p *CfLogger) Print(v ...interface{}) {
	if !p.writesToConsole {
		return
	}
	log.Print(v...)
}

func (p *CfLogger) Printf(format string, v ...interface{}) {
	if !p.writesToConsole {
		return
	}
	log.Printf(format, v...)
}

func (p *CfLogger) Println(v ...interface{}) {
	if !p.writesToConsole {
		return
	}
	log.Println(v...)
}

func (p *CfLogger) WritesToConsole() bool {
	return p.writesToConsole
}
