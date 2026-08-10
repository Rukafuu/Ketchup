package exec

import (
	"bytes"
	"context"
	"os/exec"
)

// CommandRunner é uma interface para execução de comandos externos
// Permite mocking em testes
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (int, []byte, error)
}

// DefaultCommandRunner é a implementação padrão usando exec.Command
type DefaultCommandRunner struct {
	Dir string
	Env []string
}

// Run executa um comando e retorna exit code, stdout e erro
func (r *DefaultCommandRunner) Run(ctx context.Context, name string, args ...string) (int, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.Dir
	
	if len(r.Env) > 0 {
		cmd.Env = append(cmd.Environ(), r.Env...)
	}
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	output := stdout.Bytes()
	if stderr.Len() > 0 {
		// Adiciona stderr ao output se houver
		output = append(output, stderr.Bytes()...)
	}
	
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	
	return exitCode, output, err
}

// RunWithInput executa um comando com stdin
func (r *DefaultCommandRunner) RunWithInput(ctx context.Context, input string, name string, args ...string) (int, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = r.Dir
	
	if len(r.Env) > 0 {
		cmd.Env = append(cmd.Environ(), r.Env...)
	}
	
	cmd.Stdin = bytes.NewReader([]byte(input))
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	
	output := stdout.Bytes()
	if stderr.Len() > 0 {
		output = append(output, stderr.Bytes()...)
	}
	
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	
	return exitCode, output, err
}

// CommandExists verifica se um comando está disponível no PATH
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
