package adapters

// GenericMCP is a fallback for any MCP-aware agent that doesn't have a
// dedicated adapter. Emits a portable MCP server manifest the user wires up
// manually.
type GenericMCP struct{}

func (GenericMCP) ID() string                          { return "generic-mcp" }
func (GenericMCP) DisplayName() string                 { return "Generic MCP" }
func (GenericMCP) Detect() bool                        { return true } // always available as fallback
func (GenericMCP) Connect(opts ConnectOptions) error   { return ErrNotImplemented }
func (GenericMCP) Disconnect() error                   { return ErrNotImplemented }
