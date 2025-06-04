package application

import (
	"context"
	"errors"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

// Common errors
var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// PerformanceOptimizer contains advanced performance optimization features
type PerformanceOptimizer struct {
	config           *PerformanceConfig
	objectPools      map[string]*sync.Pool
	preAllocatedMaps sync.Map
	gcOptimizer      *GCOptimizer
	cpuOptimizer     *CPUOptimizer
	memoryOptimizer  *MemoryOptimizer
}

// PerformanceConfig defines performance optimization settings
type PerformanceConfig struct {
	EnableObjectPooling   bool `json:"enable_object_pooling"`
	PoolSize              int  `json:"pool_size"`
	EnablePreallocation   bool `json:"enable_preallocation"`
	PreallocationSize     int  `json:"preallocation_size"`
	EnableGCOptimization  bool `json:"enable_gc_optimization"`
	GCTargetPercent       int  `json:"gc_target_percent"`
	EnableCPUOptimization bool `json:"enable_cpu_optimization"`
	MaxCPUUsage           int  `json:"max_cpu_usage"`
	MemoryBallastMB       int  `json:"memory_ballast_mb"`
	EnableBatchProcessing bool `json:"enable_batch_processing"`
	BatchSize             int  `json:"batch_size"`
	BatchTimeout          int  `json:"batch_timeout_ms"`
}

// NewPerformanceOptimizer creates a new performance optimizer
func NewPerformanceOptimizer(config *PerformanceConfig) *PerformanceOptimizer {
	po := &PerformanceOptimizer{
		config:      config,
		objectPools: make(map[string]*sync.Pool),
	}

	if config.EnableGCOptimization {
		po.gcOptimizer = NewGCOptimizer(config.GCTargetPercent)
	}

	if config.EnableCPUOptimization {
		po.cpuOptimizer = NewCPUOptimizer(config.MaxCPUUsage)
	}

	po.memoryOptimizer = NewMemoryOptimizer(config.MemoryBallastMB)

	return po
}

// Start starts all optimizers
func (po *PerformanceOptimizer) Start() {
	if po.gcOptimizer != nil {
		po.gcOptimizer.Start()
	}
	if po.cpuOptimizer != nil {
		po.cpuOptimizer.Start()
	}
	po.memoryOptimizer.Start()
}

// Stop stops all optimizers
func (po *PerformanceOptimizer) Stop() {
	if po.gcOptimizer != nil {
		po.gcOptimizer.Stop()
	}
	if po.cpuOptimizer != nil {
		po.cpuOptimizer.Stop()
	}
	po.memoryOptimizer.Stop()
}

// GetObjectPool returns or creates an object pool for a given type
func (po *PerformanceOptimizer) GetObjectPool(poolName string, newFunc func() interface{}) *sync.Pool {
	if pool, exists := po.objectPools[poolName]; exists {
		return pool
	}

	pool := &sync.Pool{
		New: newFunc,
	}
	po.objectPools[poolName] = pool
	return pool
}

// GCOptimizer optimizes garbage collection performance
type GCOptimizer struct {
	targetPercent  int
	stopChan       chan bool
	monitoringDone sync.WaitGroup
}

// NewGCOptimizer creates a new GC optimizer
func NewGCOptimizer(targetPercent int) *GCOptimizer {
	return &GCOptimizer{
		targetPercent: targetPercent,
		stopChan:      make(chan bool),
	}
}

// Start begins GC optimization
func (gc *GCOptimizer) Start() {
	debug.SetGCPercent(gc.targetPercent)

	gc.monitoringDone.Add(1)
	go gc.monitorGC()
}

// Stop stops GC optimization
func (gc *GCOptimizer) Stop() {
	close(gc.stopChan)
	gc.monitoringDone.Wait()
}

func (gc *GCOptimizer) monitorGC() {
	defer gc.monitoringDone.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastGCStats runtime.MemStats
	runtime.ReadMemStats(&lastGCStats)

	for {
		select {
		case <-gc.stopChan:
			return
		case <-ticker.C:
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)

			// Adaptive GC tuning based on allocation rate
			allocRate := float64(stats.TotalAlloc-lastGCStats.TotalAlloc) / 10.0 // per second
			if allocRate > 100*1024*1024 {                                       // 100MB/s
				debug.SetGCPercent(gc.targetPercent - 10) // More aggressive
			} else if allocRate < 10*1024*1024 { // 10MB/s
				debug.SetGCPercent(gc.targetPercent + 10) // Less aggressive
			}

			lastGCStats = stats
		}
	}
}

// CPUOptimizer optimizes CPU usage
type CPUOptimizer struct {
	maxUsage       int
	stopChan       chan bool
	monitoringDone sync.WaitGroup
	currentUsage   int64
}

// NewCPUOptimizer creates a new CPU optimizer
func NewCPUOptimizer(maxUsage int) *CPUOptimizer {
	return &CPUOptimizer{
		maxUsage: maxUsage,
		stopChan: make(chan bool),
	}
}

// Start begins CPU optimization
func (co *CPUOptimizer) Start() {
	co.monitoringDone.Add(1)
	go co.monitorCPU()
}

// Stop stops CPU optimization
func (co *CPUOptimizer) Stop() {
	close(co.stopChan)
	co.monitoringDone.Wait()
}

func (co *CPUOptimizer) monitorCPU() {
	defer co.monitoringDone.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-co.stopChan:
			return
		case <-ticker.C:
			currentUsage := atomic.LoadInt64(&co.currentUsage)
			if currentUsage > int64(co.maxUsage) {
				// Reduce CPU usage by adjusting GOMAXPROCS
				newMaxProcs := runtime.GOMAXPROCS(0) - 1
				if newMaxProcs > 0 {
					runtime.GOMAXPROCS(newMaxProcs)
				}
			}
		}
	}
}

// MemoryOptimizer optimizes memory usage
type MemoryOptimizer struct {
	ballastMB      int
	ballast        []byte
	stopChan       chan bool
	monitoringDone sync.WaitGroup
}

// NewMemoryOptimizer creates a new memory optimizer
func NewMemoryOptimizer(ballastMB int) *MemoryOptimizer {
	mo := &MemoryOptimizer{
		ballastMB: ballastMB,
		stopChan:  make(chan bool),
	}

	if ballastMB > 0 {
		mo.ballast = make([]byte, ballastMB*1024*1024)
	}

	return mo
}

// Start begins memory optimization
func (mo *MemoryOptimizer) Start() {
	mo.monitoringDone.Add(1)
	go mo.monitorMemory()
}

// Stop stops memory optimization
func (mo *MemoryOptimizer) Stop() {
	close(mo.stopChan)
	mo.monitoringDone.Wait()
}

func (mo *MemoryOptimizer) monitorMemory() {
	defer mo.monitoringDone.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mo.stopChan:
			return
		case <-ticker.C:
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)

			// Force GC if memory usage is too high
			if stats.HeapAlloc > 2*1024*1024*1024 { // 2GB
				runtime.GC()
			}
		}
	}
}

// BatchProcessor processes operations in batches for better performance
type BatchProcessor struct {
	batchSize     int
	batchTimeout  time.Duration
	processorFunc func([]interface{}) error
	items         []interface{}
	mutex         sync.Mutex
	timer         *time.Timer
	stopChan      chan bool
	processDone   sync.WaitGroup
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, batchTimeout time.Duration, processorFunc func([]interface{}) error) *BatchProcessor {
	bp := &BatchProcessor{
		batchSize:     batchSize,
		batchTimeout:  batchTimeout,
		processorFunc: processorFunc,
		items:         make([]interface{}, 0, batchSize),
		stopChan:      make(chan bool),
	}

	bp.processDone.Add(1)
	go bp.processLoop()

	return bp
}

// Add adds an item to the batch
func (bp *BatchProcessor) Add(item interface{}) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	bp.items = append(bp.items, item)

	if len(bp.items) >= bp.batchSize {
		bp.processBatch()
	} else if bp.timer == nil {
		bp.timer = time.AfterFunc(bp.batchTimeout, bp.timeoutProcess)
	}
}

// Stop stops the batch processor
func (bp *BatchProcessor) Stop() {
	close(bp.stopChan)
	bp.processDone.Wait()
}

func (bp *BatchProcessor) processLoop() {
	defer bp.processDone.Done()

	ticker := time.NewTicker(bp.batchTimeout)
	defer ticker.Stop()

	for {
		select {
		case <-bp.stopChan:
			// Process any remaining items
			bp.mutex.Lock()
			if len(bp.items) > 0 {
				bp.processBatch()
			}
			bp.mutex.Unlock()
			return
		case <-ticker.C:
			bp.timeoutProcess()
		}
	}
}

func (bp *BatchProcessor) timeoutProcess() {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if len(bp.items) > 0 {
		bp.processBatch()
	}
}

func (bp *BatchProcessor) processBatch() {
	if len(bp.items) == 0 {
		return
	}

	itemsCopy := make([]interface{}, len(bp.items))
	copy(itemsCopy, bp.items)
	bp.items = bp.items[:0] // Reset slice but keep capacity

	if bp.timer != nil {
		bp.timer.Stop()
		bp.timer = nil
	}

	// Process in goroutine to avoid blocking
	go func() {
		if err := bp.processorFunc(itemsCopy); err != nil {
			// Log error or handle as appropriate
		}
	}()
}

// HighPerformanceCommandMiddleware provides optimized middleware for high-throughput scenarios
func HighPerformanceCommandMiddleware(optimizer *PerformanceOptimizer) CommandMiddleware {
	// Get or create object pools
	contextPool := optimizer.GetObjectPool("context", func() interface{} {
		return make(map[string]interface{})
	})

	return func(next CommandHandler) CommandHandler {
		return CommandHandlerFunc(func(ctx context.Context, cmd Command) (interface{}, error) {
			// Use pooled objects for temporary data
			tempData := contextPool.Get().(map[string]interface{})
			defer func() {
				// Clear and return to pool
				for k := range tempData {
					delete(tempData, k)
				}
				contextPool.Put(tempData)
			}()

			// Execute with optimizations
			return next.Handle(ctx, cmd)
		})
	}
}

// HighPerformanceQueryMiddleware provides optimized middleware for high-throughput query scenarios
func HighPerformanceQueryMiddleware(optimizer *PerformanceOptimizer) QueryMiddleware {
	// Pre-allocated cache for query results
	if optimizer.config.EnablePreallocation {
		preAllocatedCache := make(map[string]interface{}, optimizer.config.PreallocationSize)
		optimizer.preAllocatedMaps.Store("query_cache", preAllocatedCache)
	}

	return func(next QueryHandler) QueryHandler {
		return QueryHandlerFunc(func(ctx context.Context, query Query) (interface{}, error) {
			// Use pre-allocated structures when possible
			if cache, ok := optimizer.preAllocatedMaps.Load("query_cache"); ok {
				cacheMap := cache.(map[string]interface{})

				// Quick cache lookup without expensive serialization
				if result, exists := cacheMap[query.QueryName()]; exists {
					return result, nil
				}
			}

			// Execute query
			result, err := next.Handle(ctx, query)

			// Cache result if successful and caching is enabled
			if err == nil && optimizer.config.EnablePreallocation {
				if cache, ok := optimizer.preAllocatedMaps.Load("query_cache"); ok {
					cacheMap := cache.(map[string]interface{})
					cacheMap[query.QueryName()] = result
				}
			}

			return result, err
		})
	}
}

// ConcurrencyOptimizer optimizes concurrent access patterns
type ConcurrencyOptimizer struct {
	shardCount   int
	shards       []shardedMap
	loadBalancer *LoadBalancer
	rateLimiter  *AdaptiveRateLimiter
}

type shardedMap struct {
	data  map[string]interface{}
	mutex sync.RWMutex
}

// NewConcurrencyOptimizer creates a new concurrency optimizer
func NewConcurrencyOptimizer(shardCount int) *ConcurrencyOptimizer {
	co := &ConcurrencyOptimizer{
		shardCount: shardCount,
		shards:     make([]shardedMap, shardCount),
	}

	// Initialize shards
	for i := 0; i < shardCount; i++ {
		co.shards[i] = shardedMap{
			data: make(map[string]interface{}),
		}
	}

	co.loadBalancer = NewLoadBalancer(shardCount)
	co.rateLimiter = NewAdaptiveRateLimiter()

	return co
}

// Get retrieves a value from the sharded map
func (co *ConcurrencyOptimizer) Get(key string) (interface{}, bool) {
	shard := co.getShard(key)
	shard.mutex.RLock()
	defer shard.mutex.RUnlock()

	value, exists := shard.data[key]
	return value, exists
}

// Set stores a value in the sharded map
func (co *ConcurrencyOptimizer) Set(key string, value interface{}) {
	shard := co.getShard(key)
	shard.mutex.Lock()
	defer shard.mutex.Unlock()

	shard.data[key] = value
}

func (co *ConcurrencyOptimizer) getShard(key string) *shardedMap {
	hash := fnv32Hash(key)
	return &co.shards[hash%uint32(co.shardCount)]
}

// LoadBalancer distributes load across multiple workers
type LoadBalancer struct {
	workers     []chan interface{}
	workerCount int
	roundRobin  int64
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(workerCount int) *LoadBalancer {
	lb := &LoadBalancer{
		workers:     make([]chan interface{}, workerCount),
		workerCount: workerCount,
	}

	for i := 0; i < workerCount; i++ {
		lb.workers[i] = make(chan interface{}, 100) // Buffered channels
	}

	return lb
}

// Submit submits work to the least loaded worker
func (lb *LoadBalancer) Submit(work interface{}) {
	// Round-robin distribution
	workerIndex := atomic.AddInt64(&lb.roundRobin, 1) % int64(lb.workerCount)

	select {
	case lb.workers[workerIndex] <- work:
		// Work submitted successfully
	default:
		// Worker is busy, try next worker
		nextWorker := (workerIndex + 1) % int64(lb.workerCount)
		select {
		case lb.workers[nextWorker] <- work:
			// Work submitted to next worker
		default:
			// All workers busy, process synchronously
			// processWork(work)
		}
	}
}

// AdaptiveRateLimiter implements adaptive rate limiting based on system load
type AdaptiveRateLimiter struct {
	baseRate    int64
	currentRate int64
	burstSize   int64
	tokens      int64
	lastRefill  time.Time
	mutex       sync.Mutex
	loadMonitor *LoadMonitor
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter() *AdaptiveRateLimiter {
	now := time.Now()
	arl := &AdaptiveRateLimiter{
		baseRate:    1000,
		currentRate: 1000,
		burstSize:   100,
		tokens:      100,
		lastRefill:  now,
		loadMonitor: NewLoadMonitor(),
	}

	return arl
}

// Allow checks if an operation is allowed
func (arl *AdaptiveRateLimiter) Allow() bool {
	arl.mutex.Lock()
	defer arl.mutex.Unlock()

	now := time.Now()

	// Refill tokens based on current rate
	elapsed := now.Sub(arl.lastRefill)
	tokensToAdd := int64(elapsed.Seconds() * float64(arl.currentRate))

	arl.tokens += tokensToAdd
	if arl.tokens > arl.burstSize {
		arl.tokens = arl.burstSize
	}

	arl.lastRefill = now

	// Adapt rate based on system load
	load := arl.loadMonitor.GetCurrentLoad()
	if load > 0.8 {
		arl.currentRate = int64(float64(arl.baseRate) * 0.5) // Reduce rate
	} else if load < 0.3 {
		arl.currentRate = int64(float64(arl.baseRate) * 1.5) // Increase rate
	}

	if arl.tokens > 0 {
		arl.tokens--
		return true
	}

	return false
}

// LoadMonitor monitors system load
type LoadMonitor struct {
	cpuUsage    float64
	memoryUsage float64
	mutex       sync.RWMutex
}

// NewLoadMonitor creates a new load monitor
func NewLoadMonitor() *LoadMonitor {
	lm := &LoadMonitor{}
	go lm.monitor()
	return lm
}

func (lm *LoadMonitor) monitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Monitor CPU usage
		numCPU := runtime.NumCPU()
		numGoroutines := runtime.NumGoroutine()
		cpuLoad := float64(numGoroutines) / float64(numCPU*100) // Rough estimate

		// Monitor memory usage
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		memLoad := float64(stats.HeapAlloc) / float64(stats.Sys)

		lm.mutex.Lock()
		lm.cpuUsage = cpuLoad
		lm.memoryUsage = memLoad
		lm.mutex.Unlock()
	}
}

// GetCurrentLoad returns the current system load (0.0 to 1.0)
func (lm *LoadMonitor) GetCurrentLoad() float64 {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	// Return the maximum of CPU and memory load
	if lm.cpuUsage > lm.memoryUsage {
		return lm.cpuUsage
	}
	return lm.memoryUsage
}

// Utility function for FNV-32 hash
func fnv32Hash(s string) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)

	hash := uint32(offset32)
	for _, c := range []byte(s) {
		hash ^= uint32(c)
		hash *= prime32
	}
	return hash
}

// OptimizedCommandBus extends the regular command bus with performance optimizations
type OptimizedCommandBus struct {
	*CommandBus
	optimizer            *PerformanceOptimizer
	concurrencyOptimizer *ConcurrencyOptimizer
	batchProcessor       *BatchProcessor
}

// NewOptimizedCommandBus creates a new optimized command bus
func NewOptimizedCommandBus(options CommandBusOptions, perfConfig *PerformanceConfig) *OptimizedCommandBus {
	regularBus := NewCommandBus(options)

	optimizer := NewPerformanceOptimizer(perfConfig)
	concurrencyOptimizer := NewConcurrencyOptimizer(16) // 16 shards

	var batchProcessor *BatchProcessor
	if perfConfig.EnableBatchProcessing {
		batchProcessor = NewBatchProcessor(
			perfConfig.BatchSize,
			time.Duration(perfConfig.BatchTimeout)*time.Millisecond,
			func(items []interface{}) error {
				// Batch processing logic would go here
				return nil
			},
		)
	}

	ocb := &OptimizedCommandBus{
		CommandBus:           regularBus,
		optimizer:            optimizer,
		concurrencyOptimizer: concurrencyOptimizer,
		batchProcessor:       batchProcessor,
	}

	// Start optimizers
	optimizer.Start()

	return ocb
}

// ExecuteCommandOptimized executes a command with optimizations
func (ocb *OptimizedCommandBus) ExecuteCommandOptimized(ctx context.Context, command Command) (interface{}, error) {
	// Simple rate limiting check - can be enhanced later
	// For now, just execute normally

	// Use batch processing for eligible commands
	if ocb.batchProcessor != nil && ocb.isBatchable(command) {
		ocb.batchProcessor.Add(command)
		return nil, nil // Async processing
	}

	// Use sharded execution for better concurrency
	return ocb.executeWithSharding(ctx, command)
}

func (ocb *OptimizedCommandBus) isBatchable(command Command) bool {
	// Define logic to determine if a command can be batched
	// For example, bulk operations, notifications, logging commands
	batchableCommands := map[string]bool{
		"LogEvent":      true,
		"SendEmail":     true,
		"UpdateMetrics": true,
	}

	return batchableCommands[command.CommandName()]
}

func (ocb *OptimizedCommandBus) executeWithSharding(ctx context.Context, command Command) (interface{}, error) {
	// Use the sharded map for caching handler lookups
	cacheKey := "handler_" + command.CommandName()
	if handler, exists := ocb.concurrencyOptimizer.Get(cacheKey); exists {
		return handler.(CommandHandler).Handle(ctx, command)
	}

	// Fallback to regular execution
	return ocb.CommandBus.ExecuteCommand(ctx, command)
}

// Stop stops all optimizers
func (ocb *OptimizedCommandBus) Stop() {
	ocb.optimizer.Stop()
	if ocb.batchProcessor != nil {
		ocb.batchProcessor.Stop()
	}
}
