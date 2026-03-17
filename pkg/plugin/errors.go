package plugin

import "fmt"

// ErrDuplicatePlugin is returned when a plugin with the same name
// is already registered in a given category.
type ErrDuplicatePlugin struct {
	Category string
	Name     string
}

func (e *ErrDuplicatePlugin) Error() string {
	return fmt.Sprintf("plugin: %s plugin %q already registered", e.Category, e.Name)
}

// ErrPluginNotFound is returned when a requested plugin does not exist
// in the registry.
type ErrPluginNotFound struct {
	Category string
	Name     string
}

func (e *ErrPluginNotFound) Error() string {
	return fmt.Sprintf("plugin: %s plugin %q not found", e.Category, e.Name)
}

// ErrNilPlugin is returned when nil is passed as a plugin to Register.
type ErrNilPlugin struct {
	Category string
}

func (e *ErrNilPlugin) Error() string {
	return fmt.Sprintf("plugin: cannot register nil %s plugin", e.Category)
}
