package crawling

import (
    "github.com/libp2p/go-libp2p"
    noise "github.com/libp2p/go-libp2p-noise"
    secio "github.com/libp2p/go-libp2p-secio"
    tls "github.com/libp2p/go-libp2p-tls"
    "sort"
    "fmt"
)

type SecurityConfig struct {
    tls int64
    noise int64
    secio int64
}

func Security (opts []libp2p.Option) []libp2p.Option{
	opts = append(opts, prioritizeOptions([]priorityOption{{
		priority:        200,
		defaultPriority: 200,
		opt:             libp2p.Security(tls.ID, tls.New),
	}, {
		priority:        Disabled,
		defaultPriority: 100,
		opt:             libp2p.Security(secio.ID, secio.New),
	}, {
		priority:        Disabled,
		defaultPriority: 300,
		opt:             libp2p.Security(noise.ID, noise.New),
	}}))
	return opts
}


type priorityOption struct {
	priority, defaultPriority Priority
	opt                       libp2p.Option
}

type Priority int64

const (
	DefaultPriority Priority = 0
	Disabled        Priority = -1
)

func prioritizeOptions(opts []priorityOption) libp2p.Option {
	type popt struct {
		priority int64
		opt      libp2p.Option
	}
	enabledOptions := make([]popt, 0, len(opts))
	for _, o := range opts {
		if prio, ok := o.priority.WithDefault(o.defaultPriority); ok {
			enabledOptions = append(enabledOptions, popt{
				priority: prio,
				opt:      o.opt,
			})
		}
	}
	sort.Slice(enabledOptions, func(i, j int) bool {
		return enabledOptions[i].priority > enabledOptions[j].priority
	})
	p2pOpts := make([]libp2p.Option, len(enabledOptions))
	for i, opt := range enabledOptions {
		p2pOpts[i] = opt.opt
	}
	return libp2p.ChainOptions(p2pOpts...)
}

func (p Priority) WithDefault(defaultPriority Priority) (priority int64, enabled bool) {
	switch p {
	case Disabled:
		return 0, false
	case DefaultPriority:
		switch defaultPriority {
		case Disabled:
			return 0, false
		case DefaultPriority:
			return 0, true
		default:
			if defaultPriority <= 0 {
				panic(fmt.Sprintf("invalid priority %d < 0", int64(defaultPriority)))
			}
			return int64(defaultPriority), true
		}
	default:
		if p <= 0 {
			panic(fmt.Sprintf("invalid priority %d < 0", int64(p)))
		}
		return int64(p), true
	}
}
