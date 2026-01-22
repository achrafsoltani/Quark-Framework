package quark

import (
	"errors"
	"testing"
)

func TestContainerRegisterAndGet(t *testing.T) {
	c := NewContainer()

	c.Register("greeting", func(c *Container) (interface{}, error) {
		return "Hello, World!", nil
	})

	result, err := c.Get("greeting")
	if err != nil {
		t.Errorf("Get: unexpected error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("Get: expected 'Hello, World!', got %v", result)
	}
}

func TestContainerSingleton(t *testing.T) {
	c := NewContainer()

	callCount := 0
	c.Register("counter", func(c *Container) (interface{}, error) {
		callCount++
		return callCount, nil
	})

	// First call
	result1, _ := c.Get("counter")
	// Second call
	result2, _ := c.Get("counter")

	if callCount != 1 {
		t.Errorf("expected factory to be called once, called %d times", callCount)
	}
	if result1 != result2 {
		t.Error("expected same instance on multiple gets")
	}
}

func TestContainerRegisterInstance(t *testing.T) {
	c := NewContainer()

	config := map[string]string{"env": "test"}
	c.RegisterInstance("config", config)

	result, err := c.Get("config")
	if err != nil {
		t.Errorf("Get: unexpected error: %v", err)
	}
	if result.(map[string]string)["env"] != "test" {
		t.Error("expected config instance")
	}
}

func TestContainerNotFound(t *testing.T) {
	c := NewContainer()

	_, err := c.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}

func TestContainerFactoryError(t *testing.T) {
	c := NewContainer()

	c.Register("failing", func(c *Container) (interface{}, error) {
		return nil, errors.New("factory failed")
	})

	_, err := c.Get("failing")
	if err == nil {
		t.Error("expected error from failing factory")
	}
}

func TestContainerMustGet(t *testing.T) {
	c := NewContainer()
	c.RegisterInstance("value", 42)

	result := c.MustGet("value")
	if result != 42 {
		t.Errorf("MustGet: expected 42, got %v", result)
	}
}

func TestContainerMustGetPanic(t *testing.T) {
	c := NewContainer()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet: expected panic for nonexistent service")
		}
	}()

	c.MustGet("nonexistent")
}

func TestContainerHas(t *testing.T) {
	c := NewContainer()

	c.Register("service", func(c *Container) (interface{}, error) {
		return "value", nil
	})
	c.RegisterInstance("instance", "value")

	if !c.Has("service") {
		t.Error("Has: expected true for registered factory")
	}
	if !c.Has("instance") {
		t.Error("Has: expected true for registered instance")
	}
	if c.Has("nonexistent") {
		t.Error("Has: expected false for nonexistent")
	}
}

func TestContainerReset(t *testing.T) {
	c := NewContainer()

	callCount := 0
	c.Register("service", func(c *Container) (interface{}, error) {
		callCount++
		return callCount, nil
	})

	c.Get("service") // First call
	c.Reset()
	c.Get("service") // Should call factory again

	if callCount != 2 {
		t.Errorf("expected factory to be called twice after reset, called %d times", callCount)
	}
}

func TestContainerClear(t *testing.T) {
	c := NewContainer()

	c.Register("service", func(c *Container) (interface{}, error) {
		return "value", nil
	})

	c.Clear()

	if c.Has("service") {
		t.Error("Clear: expected service to be removed")
	}
}

func TestContainerAlias(t *testing.T) {
	c := NewContainer()

	c.Register("original", func(c *Container) (interface{}, error) {
		return "value", nil
	})
	c.Alias("alias", "original")

	result, err := c.Get("alias")
	if err != nil {
		t.Errorf("Get alias: unexpected error: %v", err)
	}
	if result != "value" {
		t.Errorf("Get alias: expected 'value', got %v", result)
	}
}

func TestContainerKeys(t *testing.T) {
	c := NewContainer()

	c.Register("a", func(c *Container) (interface{}, error) { return "a", nil })
	c.Register("b", func(c *Container) (interface{}, error) { return "b", nil })
	c.RegisterInstance("c", "c")

	keys := c.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys: expected 3 keys, got %d", len(keys))
	}

	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}

	for _, expected := range []string{"a", "b", "c"} {
		if !keySet[expected] {
			t.Errorf("Keys: missing key %s", expected)
		}
	}
}

func TestContainerDependencies(t *testing.T) {
	c := NewContainer()

	c.Register("config", func(c *Container) (interface{}, error) {
		return map[string]string{"db": "postgres://localhost"}, nil
	})

	c.Register("db", func(c *Container) (interface{}, error) {
		config, err := c.Get("config")
		if err != nil {
			return nil, err
		}
		cfg := config.(map[string]string)
		return "connected to " + cfg["db"], nil
	})

	result, err := c.Get("db")
	if err != nil {
		t.Errorf("Get db: unexpected error: %v", err)
	}
	if result != "connected to postgres://localhost" {
		t.Errorf("Get db: unexpected result: %v", result)
	}
}

// Generic helpers tests

type TestService struct {
	Name string
}

func TestProvideAndResolve(t *testing.T) {
	c := NewContainer()

	Provide(c, "service", func(c *Container) (*TestService, error) {
		return &TestService{Name: "test"}, nil
	})

	result, err := Resolve[*TestService](c, "service")
	if err != nil {
		t.Errorf("Resolve: unexpected error: %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Resolve: expected name='test', got %s", result.Name)
	}
}

func TestProvideValue(t *testing.T) {
	c := NewContainer()

	service := &TestService{Name: "provided"}
	ProvideValue(c, "service", service)

	result, err := Resolve[*TestService](c, "service")
	if err != nil {
		t.Errorf("Resolve: unexpected error: %v", err)
	}
	if result != service {
		t.Error("Resolve: expected same instance")
	}
}

func TestResolveTypeMismatch(t *testing.T) {
	c := NewContainer()

	c.RegisterInstance("string", "hello")

	_, err := Resolve[int](c, "string")
	if err == nil {
		t.Error("Resolve: expected error for type mismatch")
	}
}

func TestMustResolve(t *testing.T) {
	c := NewContainer()

	ProvideValue(c, "value", 42)

	result := MustResolve[int](c, "value")
	if result != 42 {
		t.Errorf("MustResolve: expected 42, got %d", result)
	}
}

func TestMustResolvePanic(t *testing.T) {
	c := NewContainer()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustResolve: expected panic")
		}
	}()

	MustResolve[int](c, "nonexistent")
}

// Service Provider tests

type TestProvider struct {
	BaseProvider
	registerCalled bool
	bootCalled     bool
}

func (p *TestProvider) Register(c *Container) error {
	p.registerCalled = true
	c.RegisterInstance("test", "from provider")
	return nil
}

func (p *TestProvider) Boot(c *Container) error {
	p.bootCalled = true
	return nil
}

func TestServiceProvider(t *testing.T) {
	c := NewContainer()
	provider := &TestProvider{}

	err := c.RegisterProviders(provider)
	if err != nil {
		t.Errorf("RegisterProviders: unexpected error: %v", err)
	}

	if !provider.registerCalled {
		t.Error("expected Register to be called")
	}
	if !provider.bootCalled {
		t.Error("expected Boot to be called")
	}

	result, _ := c.Get("test")
	if result != "from provider" {
		t.Errorf("expected 'from provider', got %v", result)
	}
}

type FailingProvider struct {
	BaseProvider
}

func (p *FailingProvider) Register(c *Container) error {
	return errors.New("register failed")
}

func TestServiceProviderRegisterError(t *testing.T) {
	c := NewContainer()

	err := c.RegisterProviders(&FailingProvider{})
	if err == nil {
		t.Error("expected error from failing provider")
	}
}

func TestContainerConcurrency(t *testing.T) {
	c := NewContainer()

	c.Register("counter", func(c *Container) (interface{}, error) {
		return 1, nil
	})

	done := make(chan bool)
	for i := 0; i < 100; i++ {
		go func() {
			c.Get("counter")
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}
}
