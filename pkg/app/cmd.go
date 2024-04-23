package app

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"text/tabwriter"

	appspy "github.com/chentao-kernel/spycat/internal/app"
	"github.com/chentao-kernel/spycat/pkg/app/config"
	"github.com/chentao-kernel/spycat/pkg/core"
	"github.com/chentao-kernel/spycat/pkg/core/model"
	"github.com/chentao-kernel/spycat/pkg/ebpf/cpu"
	"github.com/chentao-kernel/spycat/pkg/log"
	"github.com/fatih/color"
	"github.com/pyroscope-io/pyroscope/pkg/cli"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Cmd struct {
	cfg     *config.Config
	RootCmd *cobra.Command
}

func NewCmd() *Cmd {
	var cfg config.Config
	rootCmd := NewRootCmd(&cfg)
	rootCmd.SilenceErrors = true
	return &Cmd{
		cfg:     &cfg,
		RootCmd: rootCmd,
	}
}

func newViper() *viper.Viper {
	return cli.NewViper("spycat")
}

func waitSignal(sigCh chan os.Signal) {
	select {
	case sig := <-sigCh:
		log.Loger.Info("Received signal and exit:%d", sig)
		os.Exit(-1)
	}
}

var (
	headerClr *color.Color
	itemClr   *color.Color
	descClr   *color.Color
	defClr    *color.Color
)

func SubCmdInit(cmd *Cmd) {
	subcommands := []*cobra.Command{
		newOffCpuSpyCmd(&cmd.cfg.OFFCPU),
		newOnCpuSpyCmd(&cmd.cfg.ONCPU),
		newFutexSnoopSpyCmd(&cmd.cfg.FUTEXSNOOP),
		newVersionCmd(),
	}

	for _, c := range subcommands {
		if c == nil {
			continue
		}
		addHelpSubcommand(c)
		c.HasHelpSubCommands()
		cmd.RootCmd.AddCommand(c)
	}

	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000000",
		FullTimestamp:   true,
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := f.File
			if len(filename) > 38 {
				filename = filename[38:]
			}
			return "", fmt.Sprintf(" %s:%d", filename, f.Line)
		},
	})
}

// common interface for all tools
func RunSpy(cfg interface{}, conf *appspy.Config, cb func(interface{}, chan *model.SpyEvent) core.BpfSpyer) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	spy, err := appspy.NewAppSpy(conf)
	if err != nil {
		fmt.Printf("new app spy failed:%v\n", err)
	}

	err = spy.Init(cfg)
	if err != nil {
		return fmt.Errorf("spy init failed:%v", err)
	}

	err = spy.Start()
	if err != nil {
		fmt.Printf("spy start failed:%v\n", err)
		spy.Stop()
	}

	fmt.Println("App Spy Start Success")
	receiver := spy.GetReceiver()

	spyer := cb(cfg, receiver.RcvChan())
	go func() {
		err := spyer.Start()
		if err != nil {
			log.Loger.Error("bpfspy:{%s}, start failed:%v\n", spyer.Name(), err)
		}
	}()
	fmt.Printf("trace event:%s start\n", spyer.Name())

	waitSignal(sigCh)
	return nil
}

func newFutexSnoopSpyCmd(cfg *config.FUTEXSNOOP) *cobra.Command {
	vpr := newViper()

	connectCmd := &cobra.Command{
		Use:   "futexsnoop [flags]",
		Short: "eBPF snoop user futex",
		Args:  cobra.NoArgs,

		RunE: cli.CreateCmdRunFn(cfg, vpr, func(_ *cobra.Command, _ []string) error {
			conf := &appspy.Config{
				Exporter: cfg.Exporter,
				Server:   cfg.Server,
			}
			return RunSpy(cfg, conf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				config, ok := cfg.(*config.FUTEXSNOOP)
				if ok {
					return cpu.NewFutexSnoopSession(model.FutexSnoop, config, buf)
				}
				return nil
			})
		}),
	}

	cli.PopulateFlagSet(cfg, connectCmd.Flags(), vpr)
	return connectCmd
}

// https://blog.csdn.net/xmcy001122/article/details/124616967 cobra use
func newOnCpuSpyCmd(cfg *config.ONCPU) *cobra.Command {
	vpr := newViper()

	connectCmd := &cobra.Command{
		Use:   "oncpu [flags]",
		Short: "eBPF oncpu sampling profiler",
		Args:  cobra.NoArgs,

		RunE: cli.CreateCmdRunFn(cfg, vpr, func(_ *cobra.Command, _ []string) error {
			conf := &appspy.Config{
				Exporter: cfg.Exporter,
				Server:   cfg.Server,
			}
			return RunSpy(cfg, conf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				config, ok := cfg.(*config.ONCPU)
				if ok {
					return cpu.NewOnCpuBpfSession(model.OnCpu, config, buf)
				}
				return nil
			})
		}),
	}

	cli.PopulateFlagSet(cfg, connectCmd.Flags(), vpr)
	return connectCmd
}

func newOffCpuSpyCmd(cfg *config.OFFCPU) *cobra.Command {
	vpr := newViper()

	connectCmd := &cobra.Command{
		Use:   "offcpu [flags]",
		Short: "eBPF offcpu profiler",
		Args:  cobra.NoArgs,

		RunE: cli.CreateCmdRunFn(cfg, vpr, func(_ *cobra.Command, _ []string) error {
			conf := &appspy.Config{
				Exporter: cfg.Exporter,
				Server:   cfg.Server,
			}
			return RunSpy(cfg, conf, func(cfg interface{}, buf chan *model.SpyEvent) core.BpfSpyer {
				config, ok := cfg.(*config.OFFCPU)
				if ok {
					return cpu.NewOffCpuBpfSession(model.OnCpu, config, buf)
				}
				return nil
			})
		}),
	}

	cli.PopulateFlagSet(cfg, connectCmd.Flags(), vpr)
	return connectCmd
}

func NewRootCmd(cfg *config.Config) *cobra.Command {
	vpr := newViper()
	rootCmd := &cobra.Command{
		Use: "spycat [flags] <subcommand>",
		Run: func(cmd *cobra.Command, _ []string) {
			if cfg.Version {
				printVersion(cmd)
			} else {
				printHelpMessage(cmd, nil)
			}
		},
	}

	rootCmd.SetUsageFunc(printUsageMessage)
	rootCmd.SetHelpFunc(printHelpMessage)
	cli.PopulateFlagSet(cfg, rootCmd.Flags(), vpr)
	return rootCmd
}

func printUsageMessage(cmd *cobra.Command) error {
	printHelpMessage(cmd, nil)
	return nil
}

func printHelpMessage(cmd *cobra.Command, _ []string) {
	cmd.Println(DefaultUsageFunc(cmd.Flags(), cmd))
}

func addHelpSubcommand(cmd *cobra.Command) {
	cmd.AddCommand(&cobra.Command{
		Use: "help",
		Run: func(_ *cobra.Command, _ []string) {
			printHelpMessage(cmd, nil)
		},
	})
}

func DefaultUsageFunc(sf *pflag.FlagSet, c *cobra.Command) string {
	var b strings.Builder

	if hasSubCommands(c) {
		headerClr.Fprintf(&b, "SUBCOMMANDS\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		for _, subcommand := range c.Commands() {
			if !subcommand.Hidden {
				fmt.Fprintf(tw, "  %s\t%s\n", itemClr.Sprintf(subcommand.Name()), subcommand.Short)
			}
		}
		tw.Flush()
		fmt.Fprintf(&b, "\n")
	}

	if countFlags(c.Flags()) > 0 {
		// headerClr.Fprintf(&b, "FLAGS\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 2, ' ', 0)
		fmt.Fprintf(tw, "%s\t  %s@new-line@\n", headerClr.Sprintf("FLAGS"), defClr.Sprint("DEFAULT VALUES"))

		// TODO: it would be nice to sort by how often people would use these.
		//   But for that we'd have to have a conversion from flag-set back to struct
		sf.VisitAll(func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			def := f.DefValue

			def = defClr.Sprint(def)
			// def = fmt.Sprintf("(%s)", def)
			fmt.Fprintf(tw, "  %s\t%s", itemClr.Sprintf("--"+f.Name), def)
			if f.Usage != "" {
				fmt.Fprintf(tw, "@new-line@    ")
				descClr.Fprint(tw, f.Usage)
			}
			descClr.Fprint(tw, "@new-line@")
			fmt.Fprint(tw, "\n")
		})
		tw.Flush()
	}

	if hasSubCommands(c) {
		b.WriteString("Run 'Spycat SUBCOMMAND --help' for more information on a subcommand.\n")
	}

	return strings.ReplaceAll(b.String(), "@new-line@", "\n")
}

func hasSubCommands(cmd *cobra.Command) bool {
	return cmd.HasSubCommands() && !(len(cmd.Commands()) == 1 && cmd.Commands()[0].Name() == "help")
}

func countFlags(fs *pflag.FlagSet) (n int) {
	fs.VisitAll(func(*pflag.Flag) { n++ })
	return n
}

func init() {
	headerClr = color.New(color.FgGreen)
	itemClr = color.New(color.Bold)
	descClr = color.New()
	defClr = color.New(color.FgYellow)
}
