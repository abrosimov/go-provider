package provider

type Logger interface {
	Infof(string, ...any)
	Warnf(string, ...any)
	Errorf(string, ...any)
}

type noopLogger struct{}

func (n noopLogger) Infof(msg string, args ...any) {}

func (n noopLogger) Warnf(msg string, args ...any) {}

func (n noopLogger) Errorf(msg string, args ...any) {}
