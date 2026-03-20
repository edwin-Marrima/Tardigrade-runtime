package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/edwin-Marrima/Tardigrade-runtime/proto"
)

const baseDir = "/"

type cmdServer struct {
	pb.UnimplementedCmdServerServer
}

// RunCmd streams the command's stdout+stderr line by line, then sends a final
// message with the process exit code.
func (s *cmdServer) RunCmd(req *pb.RunCmdRequest, stream pb.CmdServer_RunCmdServer) error {
	if strings.TrimSpace(req.Cmd) == "" {
		return status.Error(codes.InvalidArgument, "empty command")
	}

	parts := strings.Fields(req.Cmd)
	cmdName := parts[0]
	cmdArgs := parts[1:]

	cmd := exec.CommandContext(stream.Context(), "bash", "-c", req.Cmd)
	cmd.Env = append(os.Environ(), "PATH=/usr/local/bin:/usr/bin:/bin")
	cmd.Dir = baseDir

	log.WithFields(log.Fields{
		"cmd":  cmdName,
		"args": cmdArgs,
	}).Info("executing command")

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return status.Errorf(codes.Internal, "stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		return status.Errorf(codes.Internal, "failed to run command: %v", err)
	}

	// Stream both stdout and stderr back to the client as lines arrive.
	// Errors from Send are collected so we can still wait for the process.
	sendErr := make(chan error, 1)
	go func() {
		sendErr <- streamPipes(stream, stdoutPipe, stderrPipe)
	}()

	waitErr := cmd.Wait()
	pipeErr := <-sendErr

	if pipeErr != nil {
		return status.Errorf(codes.Internal, "stream error: %v", pipeErr)
	}

	// Determine exit code.
	exitCode := int32(0)
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = int32(exitErr.ExitCode())
		} else {
			return status.Errorf(codes.Internal, "wait error: %v", waitErr)
		}
	}

	log.WithFields(log.Fields{
		"cmd":      cmdName,
		"exitCode": exitCode,
	}).Info("command finished")

	// Final message carries the exit code.
	return stream.Send(&pb.RunCmdResponse{ExitCode: exitCode})
}

// streamPipes reads stdout and stderr concurrently and sends each line as a
// separate stream message.
func streamPipes(stream pb.CmdServer_RunCmdServer, stdout, stderr io.Reader) error {
	type result struct{ err error }
	ch := make(chan result, 2)

	scan := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if err := stream.Send(&pb.RunCmdResponse{Output: scanner.Text()}); err != nil {
				ch <- result{err}
				return
			}
		}
		ch <- result{scanner.Err()}
	}

	go scan(stdout)
	go scan(stderr)

	for i := 0; i < 2; i++ {
		if r := <-ch; r.err != nil {
			return r.err
		}
	}
	return nil
}

func runServer(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	srv := grpc.NewServer()
	pb.RegisterCmdServerServer(srv, &cmdServer{})

	go func() {
		log.WithField("port", port).Info("starting cmdserver")
		if err := srv.Serve(lis); err != nil {
			log.WithError(err).Fatal("server error")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	sig := <-stop
	log.WithField("signal", sig).Info("shutting down")
	srv.GracefulStop()
	return nil
}

func newRootCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "cmdserver",
		Short: "gRPC server that executes shell commands and streams output",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(port)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")

	return cmd
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}
