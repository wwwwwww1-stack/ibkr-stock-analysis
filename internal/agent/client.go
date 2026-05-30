package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"ibkr-stock-analysis/internal/domain"
)

type Client interface {
	Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error)
}

type MockClient struct {
	mu      sync.Mutex
	outputs map[string]domain.AgentOutput
	errors  map[string]error
}

func NewMockClient() *MockClient {
	return &MockClient{
		outputs: make(map[string]domain.AgentOutput),
		errors:  make(map[string]error),
	}
}

func (m *MockClient) SetOutput(symbol string, output domain.AgentOutput) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.outputs[strings.ToUpper(strings.TrimSpace(symbol))] = output
}

func (m *MockClient) SetError(symbol string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[strings.ToUpper(strings.TrimSpace(symbol))] = err
}

func (m *MockClient) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	if err := ctx.Err(); err != nil {
		return domain.AgentOutput{}, err
	}
	symbol := strings.ToUpper(strings.TrimSpace(input.Symbol))
	m.mu.Lock()
	defer m.mu.Unlock()
	if err, ok := m.errors[symbol]; ok {
		return domain.AgentOutput{}, err
	}
	output, ok := m.outputs[symbol]
	if !ok {
		return domain.AgentOutput{}, fmt.Errorf("no mock output for %s", symbol)
	}
	return output, nil
}

type ProcessClient struct {
	Command string
	Args    []string
}

func NewProcessClient(workerPath string) *ProcessClient {
	return &ProcessClient{
		Command: "node",
		Args:    []string{workerPath},
	}
}

func (p *ProcessClient) Analyze(ctx context.Context, input domain.AgentInput) (domain.AgentOutput, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		return domain.AgentOutput{}, err
	}
	cmd := exec.CommandContext(ctx, p.Command, p.Args...)
	cmd.Stdin = bytes.NewReader(append(payload, '\n'))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return domain.AgentOutput{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return domain.AgentOutput{}, err
	}
	scanner := bufio.NewScanner(&stdout)
	if !scanner.Scan() {
		if scanner.Err() != nil {
			return domain.AgentOutput{}, scanner.Err()
		}
		return domain.AgentOutput{}, errors.New("agent worker returned no response")
	}
	return DecodeResponse(scanner.Bytes())
}

type workerResponse struct {
	OK     bool               `json:"ok"`
	Result domain.AgentOutput `json:"result"`
	Error  string             `json:"error"`
}

func DecodeResponse(data []byte) (domain.AgentOutput, error) {
	var response workerResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return domain.AgentOutput{}, err
	}
	if !response.OK {
		if strings.TrimSpace(response.Error) == "" {
			return domain.AgentOutput{}, errors.New("agent worker failed")
		}
		return domain.AgentOutput{}, errors.New(response.Error)
	}
	if err := response.Result.Validate(); err != nil {
		return domain.AgentOutput{}, err
	}
	return response.Result, nil
}
