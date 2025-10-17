# Proxmox VM Manager - Modular Architecture

## Overview

The Proxmox VM Manager has been refactored into a modular architecture to improve maintainability, testability, and code organization. Each module has a specific responsibility and can be developed and tested independently.

## Module Structure

```
proxmox-vm-manager-modular/
├── main.py                     # Main entry point
├── vm_manager.py               # Main orchestrator class
├── proxmox_api.py             # Proxmox API client
├── vm_operations.py           # VM management operations
├── vm_selector.py             # VM selection and filtering
├── backup_operations.py       # Backup operations
├── storage_operations.py      # Storage operations
├── snapshot_integration.py    # Snapshot integration
├── bulk_operations.py         # Bulk operation management
└── MODULE_OVERVIEW.md         # This file
```

## Module Descriptions

### 1. `main.py`
- **Purpose**: Entry point for the application
- **Responsibilities**: 
  - Command line argument parsing
  - Application initialization
  - Error handling at the top level
  - Complete CLI interface for VM operations (start, stop, shutdown)
  - Comprehensive backup management commands (backup, list-backups, restore, delete-backups)
  - Batch mode support with safety confirmations

### 2. `vm_manager.py`
- **Purpose**: Main orchestrator that coordinates all operations
- **Responsibilities**:
  - Initialize and manage all component modules
  - Provide unified interface for VM, backup, storage, and snapshot operations
  - Handle interactive menus and user workflows (13 menu options)
  - Delegate operations to appropriate specialized modules
  - Interactive backup deletion interface (specific, pattern-based, cleanup)
  - Enhanced backup selection with range support and safety confirmations

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
  - VM lifecycle operations (start, stop, shutdown)
  - Node management and VM location finding
  - Task monitoring and progress tracking

### 5. `vm_selector.py`
- **Purpose**: Flexible VM selection and filtering
- **Responsibilities**:
  - Parse various VM selection formats (IDs, names, patterns, ranges)
  - Interactive VM selection interface
  - VM identifier resolution and validation
  - Selection help and documentation

### 6. `backup_operations.py`
- **Purpose**: VM backup lifecycle management
- **Responsibilities**:
  - Create VM backups with different modes (snapshot, suspend, stop)
  - List and display backup information with detailed formatting
  - Restore VMs from backups with protection handling
  - **Delete individual backups** with progress monitoring
  - **Bulk backup deletion** with concurrent processing
  - **Pattern-based deletion** with wildcard support (e.g., '*2024*')
  - **Automated cleanup** of old backups with retention policies
  - Safety confirmations and batch mode support for all operations

### 7. `storage_operations.py`
- **Purpose**: Storage discovery and management
- **Responsibilities**:
  - Discover VM disk and backup-capable storages
  - Display storage information
  - Validate storage space availability
  - Interactive storage selection

### 8. `snapshot_integration.py`
- **Purpose**: Integration with existing snapshot management functionality
- **Responsibilities**:
  - Bridge to proxmox-snapshot-manager modules
  - Unified snapshot operations interface
  - Snapshot listing, creation, rollback, and deletion
  - Bulk snapshot operations

### 9. `bulk_operations.py`
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

The original monolithic script has been refactored into this modular architecture to improve maintainability, testability, and code organization. The new modular version provides the same functionality with improved organization:

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

### Complete Backup Management (CRUD Operations):
The modular architecture now provides comprehensive backup lifecycle management:
- **Create**: Multiple backup modes with storage selection
- **Read**: List and display backups with detailed information
- **Update**: Restore VMs from backups with protection handling
- **Delete**: Comprehensive deletion capabilities with multiple modes:
  - Specific backup deletion using volid format
  - Pattern-based deletion with wildcard support
  - Automated cleanup with retention policies
  - Bulk operations with progress tracking

## Usage

### Basic Usage
```bash
python3 main.py
```

### With Help
```bash
python3 main.py --help
```

### Environment Variables (for authentication)
```bash
export PVE_HOST=your-proxmox-host
export PVE_USER=username@realm
export PVE_TOKEN_NAME=token-name
export PVE_TOKEN_VALUE=token-value
```

### What is volid?
**volid** (Volume Identifier) is Proxmox's unique identifier for storage objects.
- **Format**: `<STORAGE_ID>:<CONTENT_TYPE>/<PATH>`
- **Examples**: 
  - File-based backup: `local:backup/vzdump-qemu-7303-2025_08_06.vma.zst`
  - PBS backup: `backup-pbs:backup/vm/7303/2025-08-05T12:16:44Z`
- **Usage**: Required for backup deletion operations to ensure correct storage targeting

### Backup Management Examples
```bash
# Create backup
python3 main.py backup --vmid 7303 --storage local-zfs --mode snapshot --batch

# List backups to see volid format
python3 main.py list-backups --vmid 7303

# Delete specific backup (use full volid from list-backups)
python3 main.py delete-backups --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --batch --yes

# Delete by pattern
python3 main.py delete-backups --vmid 7303 --pattern "*2024*" --batch

# Cleanup old backups
python3 main.py delete-backups --vmid 7303 --cleanup --keep-count 5 --max-age-days 30 --batch --yes

# Restore from backup
python3 main.py restore --vmid 7303 --backup-file "local:backup/vzdump-qemu-7303-2025_08_06.vma.zst" --batch --yes
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

## Recent Enhancements

### Completed Features:
- ✅ **Complete Backup Management**: Full CRUD operations for backup lifecycle
- ✅ **Advanced Deletion Capabilities**: Pattern-based, bulk, and automated cleanup
- ✅ **Enhanced CLI Interface**: Comprehensive command-line options with batch mode
- ✅ **Safety Features**: Multi-level confirmations and validation

## Future Enhancements

The modular architecture enables several future improvements:

1. **Configuration Management**: Centralized configuration module
2. **Logging System**: Structured logging across all modules
3. **Plugin System**: Load additional functionality at runtime
4. **REST API**: Expose functionality via web API
5. **Web Interface**: Browser-based management interface
6. **Automated Testing**: Comprehensive test suite for all modules
7. **Backup Scheduling**: Automated backup scheduling and management
8. **Cross-cluster Operations**: Multi-cluster backup synchronization
