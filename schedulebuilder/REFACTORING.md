# Schedule Builder Refactoring

## Summary

Refactored the monolithic `main.go` (400+ lines) into 7 focused, maintainable files.

## File Structure

```
schedulebuilder/
├── main.go          (24 lines) - Entry point, program initialization
├── model.go         (75 lines) - Root model, screen state machine
├── dir_input.go     (62 lines) - Directory input screen
├── main_screen.go   (166 lines) - Main selection interface
├── search.go        (154 lines) - Search box with history & Levenshtein
├── columns.go       (63 lines) - Column state & selection logic
└── scanner.go       (35 lines) - Media file scanning
```

## Key Improvements

### 1. **Separation of Concerns**
- Each file has a single responsibility
- Screen logic separated from data management
- Search functionality isolated and reusable

### 2. **Type Safety**
- Replaced string map keys (`"0:1"`) with proper `Column` type
- Clear state machine with `screenState` enum
- Structured selection tracking per column

### 3. **Encapsulation**
```go
// Before: direct field manipulation
m.searchInput.SetValue()
m.historyIdx++

// After: methods with clear intent
searchBox.activate()
searchBox.prevHistory()
searchBox.commit()
```

### 4. **Testability**
- Pure functions (scanner, levenshtein, sortByLevenshtein)
- Isolated components (SearchBox, Column)
- No global state

### 5. **Maintainability**
- Easy to locate bugs (smaller files)
- Simple to add features (extend specific components)
- Clear data flow through the app

## Component Details

### `Column` Type
```go
type Column struct {
    items    []string
    cursor   int
    selected map[int]struct{}
}
```
Methods: `setItems`, `moveCursor`, `toggleSelection`, `isSelected`, `getSelected`

### `SearchBox` Component
```go
type SearchBox struct {
    input      textinput.Model
    history    []string
    historyIdx int
    active     bool
}
```
Methods: `activate`, `deactivate`, `commit`, `prevHistory`, `nextHistory`

### State Machine
```go
type screenState int

const (
    screenDirInput screenState = iota
    screenMain
)
```

## Benefits

✅ **80% reduction in main.go** (400+ → 24 lines)  
✅ **Clear boundaries** between UI components  
✅ **Easy to test** individual components  
✅ **Simple to extend** (add new screens/features)  
✅ **Better error isolation**  
✅ **Maintains all original functionality**  

## Philosophy Alignment

✓ **"least amount of changes"** - Same behavior, better structure  
✓ **"no fancy UI"** - Still a simple TUI, just organized  
✓ **Simple & maintainable** - Easier to debug and extend  

---

All original functionality preserved. Code compiles and runs identically.
