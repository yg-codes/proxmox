# Proxmox Snapshot Manager - Modular Architecture

## Overview

The Proxmox Snapshot Manager has been refactored into a modular architecture to improve maintainability, testability, and code organization. Each module has a specific responsibility and can be developed and tested independently.

## Module Structure

```
proxmox-snapshot-manager/
├── main.py                     # Main entry point
├── snapshot_manager.py         # Main orchestrator class
├── proxmox_api.py             # Proxmox API client
├── vm_operations.py           # VM management operations
├── vm_selector.py             # VM selection and filtering
├── snapshot_operations.py     # Snapshot CRUD operations
├── bulk_operations.py         # Bulk operation management
├── pve_snapshot_manager.py    # Original monolithic script (backup)
└── MODULE_OVERVIEW.md         # This file
```

## Module Descriptions

### 1. `main.py`
- **Purpose**: Entry point for the application
- **Responsibilities**: 
  - Command line argument parsing
  - Application initialization
  - Error handling at the top level

### 2. `snapshot_manager.py`
- **Purpose**: Main orchestrator that coordinates all operations
- **Responsibilities**:
  - Initialize and manage all component modules
  - Provide unified interface for VM and snapshot operations
  - Handle interactive menus and user workflows
  - Delegate operations to appropriate specialized modules

### 3. `proxmox_api.py`
- **Purpose**: Low-level Proxmox API communication
- **Responsibilities**:
  - Handle authentication (password and token-based)
  - Make HTTP requests to Proxmox API
  - Error handling for API communication
  - Session management

### 4. `vm_operations.py`
- **Purpose**: Virtual machine management operations
- **Responsibilities**:
  - VM discovery and status checking
  - VM information retrieval
  - VM lifecycle operations (start, stop)
  - Node management and VM location finding
  - Task monitoring and progress tracking

### 5. `vm_selector.py`
- **Purpose**: Flexible VM selection and filtering
- **Responsibilities**:
  - Parse various VM selection formats (IDs, names, patterns, ranges)
  - Interactive VM selection interface
  - VM identifier resolution and validation
  - Selection help and documentation

### 6. `snapshot_operations.py`
- **Purpose**: Snapshot lifecycle management
- **Responsibilities**:
  - Create snapshots with intelligent naming
  - Delete and rollback snapshots
  - List and display snapshot information
  - Snapshot validation and verification
  - VMstate handling

### 7. `bulk_operations.py`
- **Purpose**: Concurrent bulk operations management
- **Responsibilities**:
  - Manage concurrent operations with thread pools
  - Progress tracking and reporting
  - Result aggregation and summary reporting
  - Error handling for bulk operations

## Key Features of the Modular Design

### 1. Separation of Concerns
- Each module has a single, well-defined responsibility
- Business logic is separated from API communication
- UI/interaction logic is separated from core operations

### 2. Dependency Injection
- Components are injected into modules that need them
- Easier to test individual components in isolation
- Flexible configuration and component swapping

### 3. Error Handling
- Consistent error handling patterns across modules
- Centralized error types and messaging
- Graceful degradation when components fail

### 4. Extensibility
- Easy to add new operations without modifying existing code
- Plugin-like architecture for new features
- Clear interfaces between modules

## Migration from Monolithic Script

The original `pve_snapshot_manager.py` script has been preserved as a backup. The new modular version provides the same functionality with improved organization:

### Benefits of Modular Approach:
1. **Maintainability**: Easier to find and fix bugs in specific functionality
2. **Testability**: Each module can be unit tested independently
3. **Reusability**: Components can be reused in other projects
4. **Scalability**: Easier to add new features without affecting existing code
5. **Team Development**: Multiple developers can work on different modules

### Compatibility:
- All original functionality is preserved
- Same command-line interface and user experience
- Same configuration and authentication methods
- Existing scripts and workflows continue to work

## Usage

### Basic Usage
```bash
python3 main.py
```

### With Help
```bash
python3 main.py --help
```

### Environment Variables (same as before)
```bash
export PVE_HOST=your-proxmox-host
export PVE_USER=username@realm
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value
```

## Development

### Adding New Features
1. Identify the appropriate module for the new feature
2. Add the functionality to that module
3. Update the main manager class to expose the new feature
4. Add any necessary UI components

### Testing Individual Modules
Each module can be imported and tested independently:

```python
from vm_operations import VMOperations
from proxmox_api import ProxmoxAPI

# Test VM operations
api = ProxmoxAPI("host", "user", "password")
vm_ops = VMOperations(api)
vms = vm_ops.get_all_vms()
```

## Future Enhancements

The modular architecture enables several future improvements:

1. **Configuration Management**: Centralized configuration module
2. **Logging System**: Structured logging across all modules
3. **Plugin System**: Load additional functionality at runtime
4. **REST API**: Expose functionality via web API
5. **Web Interface**: Browser-based management interface
6. **Automated Testing**: Comprehensive test suite for all modules