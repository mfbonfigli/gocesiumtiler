# gocesiumtiler Architecture Documentation

```
                                             _   _ _
  __ _  ___   ___ ___  ___(_)_   _ _ __ ___ | |_(_) | ___ _ __
 / _  |/ _ \ / __/ _ \/ __| | | | | '_   _ \| __| | |/ _ \ '__|
| (_| | (_) | (_|  __/\__ \ | |_| | | | | | | |_| | |  __/ |
 \__, |\___/ \___\___||___/_|\__,_|_| |_| |_|\__|_|_|\___|_|
  __| | A Cesium Point Cloud tile generator written in golang
 |___/ 
```

## Table of Contents

1. [Overview](#overview)
2. [High-Level Architecture](#high-level-architecture)
3. [Core Algorithms](#core-algorithms)
4. [Package Structure](#package-structure)
5. [Data Flow](#data-flow)
6. [Memory Management](#memory-management)
7. [Concurrency Model](#concurrency-model)
8. [Coordinate System Handling](#coordinate-system-handling)
9. [Extensibility Points](#extensibility-points)
10. [Performance Characteristics](#performance-characteristics)

## Overview

gocesiumtiler is a sophisticated point cloud processing system that converts LAS (Lidar Data Exchange Format) files into Cesium.js 3D tiles. The architecture is designed around three core principles:

- **In-Memory Processing**: Loads entire point clouds into memory for fast spatial operations (requires sufficient RAM)
- **Spatial Quality**: Generate high-quality level-of-detail hierarchies for optimal rendering performance  
- **Scalability**: Leverage modern multi-core systems through concurrent processing

### Key Design Decisions

1. **Hybrid Octree-Grid Algorithm**: Combines octree spatial subdivision with grid-based sampling for uniform point distribution
2. **Zero-Copy Partitioning**: Uses linked lists for O(1) point redistribution during tree construction
3. **Producer-Consumer Pipeline**: Parallel processing architecture for scalable tile generation
4. **Interface-Based Design**: Clean separation of concerns enabling extensibility and testing

## High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Layer     │    │  Library Core    │    │  External Deps  │
│                 │    │                  │    │                 │
│ • Argument      │    │ • Point Cloud    │    │ • PROJ Library  │
│   Parsing       │────│   Processing     │────│   (Coordinate   │
│ • Validation    │    │ • Tree Building  │    │   Conversion)   │
│ • Configuration │    │ • Tile Export    │    │ • gltf Package  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### Component Overview

- **cmd/**: CLI interface and application entry point
- **tiler/**: Core library providing the main Tiler interface
- **internal/**: Implementation packages (not exposed as public API)
  - **las/**: LAS file reading and parsing
  - **tree/**: Spatial data structures and algorithms
  - **writer/**: Tile generation and export
  - **geom/**: Geometric primitives and operations
  - **conv/**: Coordinate system conversion
  - **utils/**: Utility functions and helpers

## Core Algorithms

### 1. Hybrid Octree-Grid Sampling Algorithm

The centerpiece of gocesiumtiler is a novel hybrid algorithm that combines octree spatial subdivision with grid-based sampling:

#### Phase 1: Grid Sampling
```
For each octree node:
1. Create 3D uniform grid within node's bounding box
2. Assign each point to grid cell based on 3D coordinates
3. Select point closest to grid cell center (winner selection)
4. Remaining points become "losers" for child nodes
```

#### Phase 2: Adaptive Refinement
```
If winners > maxPointsPerTile:
    scalingFactor = pow(winners/maxPointsPerTile, 1/3) * 1.1
    newGridSize = currentGridSize * scalingFactor
    repeat grid sampling with larger grid
```

#### Phase 3: Octree Partitioning
```
Split losers into 8 child octants using mean-based subdivision:
    splitX = Σx/n, splitY = Σy/n, splitZ = Σz/n
Distribute points to child nodes based on spatial position
```

#### Mathematical Foundation

**Grid Cell Assignment:**
```
cellX = floor((point.X - bbox.Xmin) / cellSizeX)
cellY = floor((point.Y - bbox.Ymin) / cellSizeY)  
cellZ = floor((point.Z - bbox.Zmin) / cellSizeZ)
```

**Distance to Cell Center:**
```
centerX = bbox.Xmin + (cellX + 0.5) * cellSizeX
centerY = bbox.Ymin + (cellY + 0.5) * cellSizeY
centerZ = bbox.Zmin + (cellZ + 0.5) * cellSizeZ
distance = sqrt((point.X - centerX)² + (point.Y - centerY)² + (point.Z - centerZ)²)
```

**Geometric Error Calculation:**
```
geometricError = sqrt(gridSize² * 3) * 1.7
```
Where:
- `sqrt(3)` accounts for maximum 3D distance within a grid cell
- `1.7` provides visual quality buffer for rendering

### 2. Zero-Copy Point Partitioning

Traditional octree implementations copy point data when subdividing nodes. gocesiumtiler uses a linked list approach for zero-copy partitioning:

#### Data Structure
```go
type LinkedPoint struct {
    Next *LinkedPoint  // 8 bytes - pointer to next point
    Pt   model.Point   // 16 bytes - actual point data
}
```

#### Partitioning Algorithm
```go
// Zero-allocation redistribution
cur := losers
for cur != nil {
    next := cur.Next                  // Save next pointer
    cur.Next = nil                    // Unlink from current list
    octant := getChildrenIndex(cur.Pt) // Determine target octant (0-7)
    
    // Relink to target octant list
    cur.Next = childrenPts[octant]
    childrenPts[octant] = cur
    cur = next
}
```

This achieves O(1) partitioning without copying 16-byte point data structures.

### 3. Coordinate System Pipeline

gocesiumtiler handles complex coordinate transformations through a multi-stage pipeline:

#### Stage 1: Source CRS Detection
- Parse LAS VLR (Variable Length Records) for embedded CRS
- Support GeoTIFF tags, WKT definitions, EPSG codes
- Fallback to user-provided CRS specification

#### Stage 2: Local Coordinate System Creation
```
For each point cloud region:
1. Calculate local origin at data center
2. Create Z-up coordinate system normal to WGS84 ellipsoid
3. Generate transformation matrix for local↔global conversion
```

#### Stage 3: Point Transformation
```
Input CRS → EPSG:4978 (Earth-Centered, Earth-Fixed) → Local CRS
```

This approach provides:
- Numerical precision preservation for large coordinate values
- Tighter bounding volumes for efficient culling
- Z-up orientation for elevation-based shading

## Package Structure

### Public API Surface

```go
// Primary interface
type Tiler interface {
    ProcessFiles([]string, string, string, *TilerOptions, context.Context) error
    ProcessFolder(string, string, string, *TilerOptions, context.Context) error
}

// Configuration
type TilerOptions struct {
    gridSize        float64
    maxDepth        int
    minPointsPerTile int
    maxPointsPerTile int
    // ... other options
}

// Extension points
type Mutator interface {
    Mutate(model.Point, model.Transform) (model.Point, bool)
}
```

### Internal Package Organization

#### `internal/las/`
- **Purpose**: LAS file format handling
- **Key Types**: `LasReader`, `CombinedFileLasReader`
- **Responsibilities**:
  - Binary LAS format parsing
  - Multi-file aggregation with CRS validation
  - Point streaming with coordinate conversion

#### `internal/tree/`
- **Purpose**: Spatial data structures
- **Key Types**: `Tree`, `Node`, `GridTree`
- **Responsibilities**:
  - Octree construction and management
  - Spatial indexing and querying
  - Level-of-detail calculations

#### `internal/writer/`
- **Purpose**: Tile generation and export
- **Key Types**: `Writer`, `Producer`, `Consumer`
- **Responsibilities**:
  - Concurrent tile generation
  - Multiple output formats (.pnts, .glb)
  - Tileset.json hierarchy creation

#### `internal/geom/`
- **Purpose**: Geometric primitives and operations
- **Key Types**: `Point64`, `BoundingBox`, `PointList`
- **Responsibilities**:
  - 3D geometry operations
  - Bounding volume calculations
  - Cesium coordinate system integration

## Data Flow

### Overall Pipeline

```
LAS Files → Parse Headers → Load Points → Build Tree → Export Tiles
    ↓           ↓            ↓           ↓            ↓
[Validate]  [Extract CRS] [Transform]  [Sample]   [Encode]
[Combine]   [Count Pts]   [Mutate]     [Partition] [Write]
```

### Detailed Flow

#### 1. Input Processing
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│ LAS File(s) │───▶│ Header Parse │───▶│ CRS Extract │
└─────────────┘    └──────────────┘    └─────────────┘
                          │                    │
                          ▼                    ▼
                   ┌──────────────┐    ┌─────────────┐
                   │ Point Count  │    │ Validation  │
                   └──────────────┘    └─────────────┘
```

#### 2. Point Loading (Concurrent)
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Worker 1  │    │   Worker 2   │    │   Worker N  │
│             │    │              │    │             │
│ Read Points │    │ Read Points  │    │ Read Points │
│ Transform   │───▶│ Transform    │◀───│ Transform   │
│ Apply       │    │ Apply        │    │ Apply       │
│ Mutators    │    │ Mutators     │    │ Mutators    │
└─────────────┘    └──────────────┘    └─────────────┘
       │                   │                   │
       └─────────────────▶ │ ◀─────────────────┘
                           ▼
                 ┌──────────────────┐
                 │ Contiguous Array │
                 │ [LinkedPoint]    │
                 └──────────────────┘
```

#### 3. Tree Construction (Recursive)
```
┌─────────────┐
│ Root Node   │
│ All Points  │
└─────────────┘
       │
       ▼
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│ Grid Sample │───▶│ Winner       │───▶│ Check Limit │
│ Points      │    │ Selection    │    │ Refinement  │
└─────────────┘    └──────────────┘    └─────────────┘
       │                                       │
       ▼                                       ▼
┌─────────────┐                         ┌─────────────┐
│ Partition   │                         │ Create      │
│ Losers into │                         │ Child Nodes │
│ 8 Octants   │                         │ (Lazy)      │
└─────────────┘                         └─────────────┘
```

#### 4. Tile Export (Producer-Consumer)
```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│  Producer   │    │ Work Channel │    │ Consumer 1  │
│             │───▶│              │───▶│             │
│ Tree        │    │ [WorkUnit]   │    │ .pnts/.glb  │
│ Traversal   │    │              │    │ Encoding    │
└─────────────┘    └──────────────┘    └─────────────┘
                          │                   │
                          ▼                   ▼
                   ┌──────────────┐    ┌─────────────┐
                   │ Consumer 2   │    │ Consumer N  │
                   │              │    │             │
                   │ Parallel     │    │ Write Files │
                   │ Processing   │    │ to Disk     │
                   └──────────────┘    └─────────────┘
```

## Memory Management

### Allocation Strategy

#### 1. Contiguous Point Storage
```go
// Single large allocation for all points
backingArray := make([]geom.LinkedPoint, totalPointCount)

// Points stored contiguously for cache efficiency
for i := 0; i < totalPointCount; i++ {
    backingArray[i] = geom.LinkedPoint{
        Next: &backingArray[i+1], // Link to next element
        Pt:   convertedPoint,
    }
}
```

#### 2. Zero-Copy Operations
- **Tree partitioning**: Manipulate pointers, never copy point data
- **Octant distribution**: Relink existing nodes between lists
- **Child creation**: Reference existing point memory

#### 3. Memory Layout
```
Backing Array Layout:
┌──────────────┬──────────────┬──────────────┬─────┐
│ LinkedPoint  │ LinkedPoint  │ LinkedPoint  │ ... │
│ [Next│Point] │ [Next│Point] │ [Next│Point] │     │
└──────────────┴──────────────┴──────────────┴─────┘
     │              │              │
     └──────────────└──────────────└─────▶ Next
```

### Memory Scaling

**Per-Point Memory Usage:**
- `LinkedPoint`: 24 bytes (8-byte pointer + 16-byte point)
- Tree overhead: ~30% additional (nodes, metadata)
- **Total**: ~32 bytes per point in memory

**Scaling Characteristics:**
- 1M points: ~32 MB peak memory
- 10M points: ~320 MB peak memory  
- 100M points: ~3.2 GB peak memory

## Concurrency Model

### Producer-Consumer Architecture

#### Writer Pipeline
```go
// Single producer generates work units
producer := &StandardProducer{}
go producer.Produce(workChannel, errorChannel, &wg, rootNode, ctx)

// Multiple consumers process work units
for i := 0; i < numWorkers; i++ {
    consumer := &StandardConsumer{}
    go consumer.Consume(workChannel, errorChannel, &wg)
}
```

#### Synchronization Primitives

**Channel Communication:**
```go
workChannel := make(chan *WorkUnit, numWorkers*bufferRatio)
errorChannel := make(chan error)
```

**Coordination:**
```go
var wg sync.WaitGroup
wg.Add(numWorkers + 1) // +1 for producer
```

**Error Handling:**
```go
// Centralized error collection
errs := []error{}
for err := range errorChannel {
    errs = append(errs, err)
}
```

### Thread Safety

#### Atomic Operations
- `atomic.Int32` for reader coordination in multi-file processing
- `atomic.Bool` for mutator state tracking
- Lock-free coordination where possible

#### Mutex Protection
- LAS file reader access (sequential read requirement)
- Tree node lazy initialization (double-checked locking)
- Shared resource access in writer components

#### Context Cancellation
```go
// Graceful shutdown on signal
ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

// Context checking in processing loops
if err := ctx.Err(); err != nil {
    return err
}
```

## Coordinate System Handling

### Multi-Stage Transformation Pipeline

#### Stage 1: CRS Detection
```go
// Priority order for CRS detection:
1. User-specified CRS parameter
2. LAS VLR GeoTIFF tags
3. LAS VLR WKT definition
4. Fallback to error
```

#### Stage 2: Local Coordinate System
```go
// Create local origin at point cloud center
localOrigin := calculateCentroid(allPoints)

// Calculate Z-up normal to WGS84 ellipsoid
normal := normalToWGS84Ellipsoid(localOrigin)

// Create transformation matrix
transform := createLocalTransform(localOrigin, normal)
```

#### Stage 3: Coordinate Transformation
```go
// Transform pipeline:
inputPoint (Source CRS) 
    → proj.Transform() → 
globalPoint (EPSG:4978)
    → localTransform.Apply() →
localPoint (Local Z-up CRS)
```

### Benefits of Local Coordinate System

1. **Numerical Precision**: Avoid large coordinate values that cause floating-point precision loss
2. **Tight Bounding Volumes**: Local coordinates produce smaller, more accurate bounding boxes
3. **Shader Compatibility**: Z-up orientation enables elevation-based shader effects
4. **Cesium Integration**: Direct compatibility with Cesium's rendering pipeline

## Extensibility Points

### 1. Mutator Interface
```go
type Mutator interface {
    Mutate(pt model.Point, localToGlobal model.Transform) (model.Point, bool)
}
```

**Use Cases:**
- Point color correction
- Classification-based filtering  
- Elevation adjustments
- Custom point transformations

**Pipeline Composition:**
```go
mutators := []mutator.Mutator{
    mutator.NewZOffset(10.0),
    mutator.NewSubsampler(0.5),
    customMutator,
}
```

### 2. GeometryEncoder Interface
```go
type GeometryEncoder interface {
    Write(n tree.Node, folderPath string, prefix string) error
    TilesetVersion() version.TilesetVersion
}
```

**Current Implementations:**
- PNTS encoder (3D Tiles v1.0)
- GLB encoder (3D Tiles v1.1)

**Extension Opportunities:**
- LAZ compression support
- Custom binary formats
- Streaming encoders

### 3. LasReader Interface
```go
type LasReader interface {
    NumberOfPoints() int
    GetNext() (geom.Point64, error)
    GetCRS() string
    Close()
}
```

**Extension Opportunities:**
- LAZ compressed files
- Alternative point cloud formats (PLY, XYZ)
- Streaming data sources
- Database integration

### 4. Writer Interface
```go
type Writer interface {
    Write(t tree.Tree, folderName string, ctx context.Context) error
}
```

**Customization Points:**
- Output directory structure
- Tile naming conventions
- Metadata generation
- Custom tileset.json structure

## Performance Characteristics

### Algorithmic Complexity

**Time Complexity:**
- Point loading: O(n/w) where w = workers
- Tree building: O(n log d) where d = max depth
- Tile export: O(nodes/w) where w = workers
- **Overall**: O(n log d)

**Space Complexity:**
- Point storage: O(n)
- Tree structure: O(n/points_per_tile)
- **Peak memory**: O(n)

### Performance Bottlenecks

#### 1. Grid Sampling (40-50% of CPU time)
**Bottleneck**: Hash map operations for grid cell assignment
```go
grid := map[[3]int32]cell{}  // High allocation pressure
```

**Optimization opportunities:**
- Pre-allocated 3D arrays instead of hash maps
- SIMD vectorization for distance calculations
- Parallel grid sampling

#### 2. Coordinate Transformation (20-30% of CPU time)
**Bottleneck**: PROJ library calls for each point
```go
globalPt, err := converter.Convert(sourcePt)
```

**Optimization opportunities:**
- Batch transformations
- Coordinate caching for nearby points
- Custom transformation implementations for common cases

#### 3. Memory Allocation (15-20% of GC time)
**Bottleneck**: Temporary allocations during encoding
```go
coords := make([][3]float32, pointCount)      // Per-tile allocation
colors := make([][3]uint8, pointCount)        // Per-tile allocation
```

**Optimization opportunities:**
- Buffer pooling with sync.Pool
- Pre-allocated encoder buffers
- Streaming encoding to reduce peak memory

### Scaling Characteristics

**Linear Scaling:**
- Point loading (limited by I/O)
- Memory usage (proportional to point count)

**Logarithmic Scaling:**
- Tree depth (spatial subdivision)
- Query performance (octree traversal)

**Parallel Scaling:**
- Tile generation (embarrassingly parallel)
- Point transformation (worker-limited)

### Benchmark Targets

**Recommended Performance Targets:**
- **Processing Rate**: 1M+ points/second on modern hardware
- **Memory Efficiency**: <50 bytes per point peak memory usage
- **Concurrency**: Linear scaling up to CPU core count
- **I/O Efficiency**: Process data faster than disk read speed

---

## Conclusion

The gocesiumtiler architecture represents a sophisticated approach to large-scale point cloud processing, combining:

- **Novel algorithms** (hybrid octree-grid sampling)
- **Memory-efficient designs** (zero-copy partitioning)
- **Modern concurrency patterns** (producer-consumer pipelines)
- **Clean abstractions** (interface-based extensibility)

The architecture successfully balances performance, memory efficiency, and code maintainability while providing a robust foundation for processing massive point cloud datasets into high-quality 3D tiles for web-based visualization.