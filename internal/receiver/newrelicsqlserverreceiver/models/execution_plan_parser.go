// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"encoding/xml"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// XML Structure for SQL Server Execution Plans
// Based on Microsoft's showplan XML schema

// ShowPlanXML represents the root element of SQL Server execution plan XML
type ShowPlanXML struct {
	XMLName       xml.Name        `xml:"ShowPlanXML"`
	Version       string          `xml:"Version,attr"`
	Build         string          `xml:"Build,attr"`
	BatchSequence []BatchSequence `xml:"BatchSequence"`
}

// BatchSequence represents a batch of SQL statements
type BatchSequence struct {
	XMLName xml.Name `xml:"BatchSequence"`
	Batch   []Batch  `xml:"Batch"`
}

// Batch represents a single batch execution
type Batch struct {
	XMLName    xml.Name     `xml:"Batch"`
	Statements []StmtSimple `xml:"Statements>StmtSimple"`
}

// StmtSimple represents a simple SQL statement
type StmtSimple struct {
	XMLName                           xml.Name  `xml:"StmtSimple"`
	StatementText                     string    `xml:"StatementText,attr"`
	StatementId                       int       `xml:"StatementId,attr"`
	StatementCompId                   int       `xml:"StatementCompId,attr"`
	StatementType                     string    `xml:"StatementType,attr"`
	RetrievedFromCache                string    `xml:"RetrievedFromCache,attr"`
	StatementSubTreeCost              string    `xml:"StatementSubTreeCost,attr"`
	StatementEstRows                  string    `xml:"StatementEstRows,attr"`
	SecurityPolicyApplied             string    `xml:"SecurityPolicyApplied,attr"`
	StatementOptmLevel                string    `xml:"StatementOptmLevel,attr"`
	QueryHash                         string    `xml:"QueryHash,attr"`
	QueryPlanHash                     string    `xml:"QueryPlanHash,attr"`
	CardinalityEstimationModelVersion string    `xml:"CardinalityEstimationModelVersion,attr"`
	QueryPlan                         QueryPlan `xml:"QueryPlan"`
}

// QueryPlan represents the execution plan details
type QueryPlan struct {
	XMLName               xml.Name        `xml:"QueryPlan"`
	DegreeOfParallelism   string          `xml:"DegreeOfParallelism,attr"`
	MemoryGrant           string          `xml:"MemoryGrant,attr"`
	CachedPlanSize        string          `xml:"CachedPlanSize,attr"`
	CompileTime           string          `xml:"CompileTime,attr"`
	CompileCPU            string          `xml:"CompileCPU,attr"`
	CompileMemory         string          `xml:"CompileMemory,attr"`
	NonParallelPlanReason string          `xml:"NonParallelPlanReason,attr"`
	RelOp                 RelOp           `xml:"RelOp"`
	ParameterList         *ParameterList  `xml:"ParameterList,omitempty"`
	QueryTimeStats        *QueryTimeStats `xml:"QueryTimeStats,omitempty"`
}

// RelOp represents a relational operator in the execution plan
type RelOp struct {
	XMLName                   xml.Name `xml:"RelOp"`
	NodeId                    int      `xml:"NodeId,attr"`
	PhysicalOp                string   `xml:"PhysicalOp,attr"`
	LogicalOp                 string   `xml:"LogicalOp,attr"`
	EstimateRows              string   `xml:"EstimateRows,attr"`
	EstimateIO                string   `xml:"EstimateIO,attr"`
	EstimateCPU               string   `xml:"EstimateCPU,attr"`
	AvgRowSize                string   `xml:"AvgRowSize,attr"`
	EstimatedTotalSubtreeCost string   `xml:"EstimatedTotalSubtreeCost,attr"`
	Parallel                  string   `xml:"Parallel,attr"`
	EstimateRebinds           string   `xml:"EstimateRebinds,attr"`
	EstimateRewinds           string   `xml:"EstimateRewinds,attr"`
	EstimatedExecutionMode    string   `xml:"EstimatedExecutionMode,attr"`

	// Child operators
	RelOp []RelOp `xml:"RelOp"`

	// Join operator specific children (for NestedLoops, Hash, Merge joins)
	OneSide   *RelOp `xml:"OneSide>RelOp,omitempty"`
	OtherSide *RelOp `xml:"OtherSide>RelOp,omitempty"`

	// Operator-specific details
	RunTimeInformation *RunTimeInformation `xml:"RunTimeInformation,omitempty"`
	MemoryFractions    *MemoryFractions    `xml:"MemoryFractions,omitempty"`
	OutputList         *OutputList         `xml:"OutputList,omitempty"` // Column references (at RelOp level)
	IndexScan          *IndexScan          `xml:"IndexScan,omitempty"`
	NestedLoops        *NestedLoops        `xml:"NestedLoops,omitempty"`
	Hash               *Hash               `xml:"Hash,omitempty"`
	Sort               *Sort               `xml:"Sort,omitempty"`
	Filter             *Filter             `xml:"Filter,omitempty"`
	ComputeScalar      *ComputeScalar      `xml:"ComputeScalar,omitempty"`
	StreamAggregate    *StreamAggregate    `xml:"StreamAggregate,omitempty"`
	Merge              *Merge              `xml:"Merge,omitempty"`
	Concatenation      *Concatenation      `xml:"Concatenation,omitempty"`
	Parallelism        *Parallelism        `xml:"Parallelism,omitempty"`
}

// RunTimeInformation contains actual execution statistics
type RunTimeInformation struct {
	XMLName           xml.Name                   `xml:"RunTimeInformation"`
	CountersPerThread []RunTimeCountersPerThread `xml:"RunTimeCountersPerThread"`
}

// RunTimeCountersPerThread contains per-thread execution counters
type RunTimeCountersPerThread struct {
	XMLName                xml.Name `xml:"RunTimeCountersPerThread"`
	Thread                 int      `xml:"Thread,attr"`
	ActualRows             string   `xml:"ActualRows,attr"`
	ActualEndOfScans       string   `xml:"ActualEndOfScans,attr"`
	ActualExecutions       string   `xml:"ActualExecutions,attr"`
	ActualExecutionMode    string   `xml:"ActualExecutionMode,attr"`
	ActualElapsedms        string   `xml:"ActualElapsedms,attr"`
	ActualCPUms            string   `xml:"ActualCPUms,attr"`
	ActualScans            string   `xml:"ActualScans,attr"`
	ActualLogicalReads     string   `xml:"ActualLogicalReads,attr"`
	ActualPhysicalReads    string   `xml:"ActualPhysicalReads,attr"`
	ActualReadAheads       string   `xml:"ActualReadAheads,attr"`
	ActualLobLogicalReads  string   `xml:"ActualLobLogicalReads,attr"`
	ActualLobPhysicalReads string   `xml:"ActualLobPhysicalReads,attr"`
	ActualLobReadAheads    string   `xml:"ActualLobReadAheads,attr"`
}

// MemoryFractions contains memory grant information
type MemoryFractions struct {
	XMLName xml.Name `xml:"MemoryFractions"`
	Input   string   `xml:"Input,attr"`
	Output  string   `xml:"Output,attr"`
}

// IndexScan contains index scan operation details
type IndexScan struct {
	XMLName       xml.Name    `xml:"IndexScan"`
	Ordered       string      `xml:"Ordered,attr"`
	ScanDirection string      `xml:"ScanDirection,attr"`
	ForcedIndex   string      `xml:"ForcedIndex,attr"`
	ForceSeek     string      `xml:"ForceSeek,attr"`
	Object        *Object     `xml:"Object,omitempty"`     // Table/Index information
	RelOp         []RelOp     `xml:"RelOp"`                // Child operators
	OutputList    *OutputList `xml:"OutputList,omitempty"` // Referenced columns
}

// NestedLoops contains nested loop join details
type NestedLoops struct {
	XMLName               xml.Name `xml:"NestedLoops"`
	Optimized             string   `xml:"Optimized,attr"`
	WithUnorderedPrefetch string   `xml:"WithUnorderedPrefetch,attr"`
	RelOp                 []RelOp  `xml:"RelOp"` // Child operators
}

// Hash contains hash operation details
type Hash struct {
	XMLName       xml.Name `xml:"Hash"`
	BitmapCreator string   `xml:"BitmapCreator,attr"`
	RelOp         []RelOp  `xml:"RelOp"` // Child operators
}

// Sort contains sort operation details
type Sort struct {
	XMLName  xml.Name `xml:"Sort"`
	Distinct string   `xml:"Distinct,attr"`
	RelOp    []RelOp  `xml:"RelOp"` // Child operators
}

// Filter contains filter operation details
type Filter struct {
	XMLName xml.Name `xml:"Filter"`
	RelOp   []RelOp  `xml:"RelOp"` // Child operators
}

// ComputeScalar contains compute scalar operation details
type ComputeScalar struct {
	XMLName xml.Name `xml:"ComputeScalar"`
	RelOp   []RelOp  `xml:"RelOp"` // Child operators
}

// StreamAggregate contains stream aggregate operation details
type StreamAggregate struct {
	XMLName xml.Name `xml:"StreamAggregate"`
	RelOp   []RelOp  `xml:"RelOp"` // Child operators
}

// Merge contains merge operation details
type Merge struct {
	XMLName   xml.Name `xml:"Merge"`
	InnerSide string   `xml:"InnerSide,attr"`
	RelOp     []RelOp  `xml:"RelOp"` // Child operators
}

// Concatenation contains concatenation operation details
type Concatenation struct {
	XMLName xml.Name `xml:"Concatenation"`
	RelOp   []RelOp  `xml:"RelOp"` // Child operators
}

// Parallelism contains parallelism operation details
type Parallelism struct {
	XMLName        xml.Name `xml:"Parallelism"`
	PartitionCount string   `xml:"PartitionCount,attr"`
	RelOp          []RelOp  `xml:"RelOp"` // Child operators
}

// ParameterList contains query parameters
type ParameterList struct {
	XMLName         xml.Name          `xml:"ParameterList"`
	ColumnReference []ColumnReference `xml:"ColumnReference"`
}

// ColumnReference represents a column or parameter reference
type ColumnReference struct {
	XMLName                xml.Name `xml:"ColumnReference"`
	Column                 string   `xml:"Column,attr"`
	ParameterCompiledValue string   `xml:"ParameterCompiledValue,attr"`
	ParameterRuntimeValue  string   `xml:"ParameterRuntimeValue,attr"`
}

// QueryTimeStats contains query execution timing statistics
type QueryTimeStats struct {
	XMLName     xml.Name `xml:"QueryTimeStats"`
	ElapsedTime string   `xml:"ElapsedTime,attr"`
	CpuTime     string   `xml:"CpuTime,attr"`
}

// Object represents table/index information in execution plan
type Object struct {
	XMLName   xml.Name `xml:"Object"`
	Database  string   `xml:"Database,attr"`
	Schema    string   `xml:"Schema,attr"`
	Table     string   `xml:"Table,attr"`
	Index     string   `xml:"Index,attr"`
	IndexKind string   `xml:"IndexKind,attr"`
}

// OutputList contains list of columns being output by an operator
type OutputList struct {
	XMLName         xml.Name          `xml:"OutputList"`
	ColumnReference []ColumnReference `xml:"ColumnReference"`
}

// DefinedValues contains computed/derived values in execution plan
type DefinedValues struct {
	XMLName         xml.Name          `xml:"DefinedValues"`
	ColumnReference []ColumnReference `xml:"ColumnReference"`
}

// ParseExecutionPlanXML parses SQL Server execution plan XML into structured data
func ParseExecutionPlanXML(planXML, queryID, planHandle string) (*ExecutionPlanAnalysis, error) {
	if planXML == "" {
		return nil, fmt.Errorf("empty execution plan XML")
	}

	var showPlan ShowPlanXML
	err := xml.Unmarshal([]byte(planXML), &showPlan)
	if err != nil {
		return nil, fmt.Errorf("failed to parse execution plan XML: %w", err)
	}

	analysis := &ExecutionPlanAnalysis{
		QueryID:        queryID,
		PlanHandle:     planHandle,
		Nodes:          []ExecutionPlanNode{},
		CollectionTime: time.Now().UTC().Format(time.RFC3339),
	}

	// Extract SQL text and plan-level information
	if len(showPlan.BatchSequence) > 0 && len(showPlan.BatchSequence[0].Batch) > 0 {
		batch := showPlan.BatchSequence[0].Batch[0]
		if len(batch.Statements) > 0 {
			stmt := batch.Statements[0]
			analysis.SQLText = stmt.StatementText

			// Parse plan-level costs and timing
			if cost, err := strconv.ParseFloat(stmt.StatementSubTreeCost, 64); err == nil {
				analysis.TotalCost = cost
			}

			// Parse query plan details
			queryPlan := stmt.QueryPlan
			analysis.CompileTime = queryPlan.CompileTime
			if cpu, err := strconv.ParseInt(queryPlan.CompileCPU, 10, 64); err == nil {
				analysis.CompileCPU = cpu
			}
			if memory, err := strconv.ParseInt(queryPlan.CompileMemory, 10, 64); err == nil {
				analysis.CompileMemory = memory
			}

			// Parse operators recursively
			nodeID := 0
			parseRelOpRecursively(&queryPlan.RelOp, analysis, queryID, planHandle, &nodeID, -1, "Root")
		}
	}

	return analysis, nil
}

// parseRelOpRecursively recursively parses relational operators in the execution plan
func parseRelOpRecursively(relOp *RelOp, analysis *ExecutionPlanAnalysis, queryID, planHandle string, nodeID *int, parentNodeID int, incomingInputType string) {
	if relOp == nil {
		return
	}

	*nodeID++
	currentNodeID := *nodeID

	// Debug logging to track recursion
	log.Printf("DEBUG: Processing node %d (parent: %d) - PhysicalOp: %s, InputType: %s, Children: %d, OneSide: %v, OtherSide: %v",
		currentNodeID, parentNodeID, relOp.PhysicalOp, incomingInputType, len(relOp.RelOp), relOp.OneSide != nil, relOp.OtherSide != nil)
	node := ExecutionPlanNode{
		QueryID:                queryID,
		PlanHandle:             planHandle,
		NodeID:                 currentNodeID,
		ParentNodeID:           parentNodeID,
		InputType:              incomingInputType,
		SQLText:                analysis.SQLText,
		PhysicalOp:             relOp.PhysicalOp,
		LogicalOp:              relOp.LogicalOp,
		EstimatedExecutionMode: relOp.EstimatedExecutionMode,
		CollectionTimestamp:    analysis.CollectionTime,
	}

	// Parse numeric values safely
	if val, err := strconv.ParseFloat(relOp.EstimateRows, 64); err == nil {
		node.EstimateRows = val
	}
	if val, err := strconv.ParseFloat(relOp.EstimateIO, 64); err == nil {
		node.EstimateIO = val
	}
	if val, err := strconv.ParseFloat(relOp.EstimateCPU, 64); err == nil {
		node.EstimateCPU = val
	}
	if val, err := strconv.ParseFloat(relOp.AvgRowSize, 64); err == nil {
		node.AvgRowSize = val
	}
	if val, err := strconv.ParseFloat(relOp.EstimatedTotalSubtreeCost, 64); err == nil {
		node.TotalSubtreeCost = val
		node.EstimatedOperatorCost = val // Operator cost is part of subtree cost
	}

	// Parse runtime information if available
	if relOp.RunTimeInformation != nil && len(relOp.RunTimeInformation.CountersPerThread) > 0 {
		counter := relOp.RunTimeInformation.CountersPerThread[0] // Take first thread's data

		if val, err := strconv.ParseFloat(counter.ActualElapsedms, 64); err == nil {
			node.TotalElapsedTime = val
			node.AvgElapsedTimeMs = val
		}
		if val, err := strconv.ParseFloat(counter.ActualCPUms, 64); err == nil {
			node.TotalWorkerTime = val
		}
		if val, err := strconv.ParseInt(counter.ActualLogicalReads, 10, 64); err == nil {
			node.TotalLogicalReads = val
		}
		if val, err := strconv.ParseInt(counter.ActualExecutions, 10, 64); err == nil {
			node.ExecutionCount = val
		}
	}

	// Extract table/index information from IndexScan operator
	if relOp.IndexScan != nil && relOp.IndexScan.Object != nil {
		obj := relOp.IndexScan.Object
		node.SchemaName = obj.Schema
		node.TableName = obj.Table
		node.IndexName = obj.Index
	}

	// Extract referenced columns from OutputList (at RelOp level for all operators)
	if relOp.OutputList != nil {
		var columns []string
		for _, colRef := range relOp.OutputList.ColumnReference {
			if colRef.Column != "" {
				columns = append(columns, colRef.Column)
			}
		}
		node.ReferencedColumns = strings.Join(columns, ", ")
	}

	// Fallback: Extract columns from IndexScan.OutputList if RelOp.OutputList is empty
	if node.ReferencedColumns == "" && relOp.IndexScan != nil && relOp.IndexScan.OutputList != nil {
		var columns []string
		for _, colRef := range relOp.IndexScan.OutputList.ColumnReference {
			if colRef.Column != "" {
				columns = append(columns, colRef.Column)
			}
		}
		node.ReferencedColumns = strings.Join(columns, ", ")
	}

	// Check for performance issues
	node.SpillOccurred = checkForSpill(relOp)
	node.NoJoinPredicate = checkForMissingJoinPredicate(relOp)

	// Set current timestamp
	node.LastExecutionTime = time.Now().UTC().Format(time.RFC3339)

	analysis.Nodes = append(analysis.Nodes, node)

	// Recursively parse child operators based on their position/role
	// Handle join operators with explicit left/right sides
	if relOp.OneSide != nil {
		parseRelOpRecursively(relOp.OneSide, analysis, queryID, planHandle, nodeID, currentNodeID, "LeftInput")
	}
	if relOp.OtherSide != nil {
		parseRelOpRecursively(relOp.OtherSide, analysis, queryID, planHandle, nodeID, currentNodeID, "RightInput")
	}

	// Handle generic child operators (for single-input operators like Compute Scalar, Filter, Sort, etc.)
	for i := range relOp.RelOp {
		parseRelOpRecursively(&relOp.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
	}

	// Handle children nested inside operator-specific structures
	if relOp.NestedLoops != nil {
		for i := range relOp.NestedLoops.RelOp {
			parseRelOpRecursively(&relOp.NestedLoops.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.Hash != nil {
		for i := range relOp.Hash.RelOp {
			parseRelOpRecursively(&relOp.Hash.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.Sort != nil {
		for i := range relOp.Sort.RelOp {
			parseRelOpRecursively(&relOp.Sort.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.IndexScan != nil {
		for i := range relOp.IndexScan.RelOp {
			parseRelOpRecursively(&relOp.IndexScan.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.Filter != nil {
		for i := range relOp.Filter.RelOp {
			parseRelOpRecursively(&relOp.Filter.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.ComputeScalar != nil {
		for i := range relOp.ComputeScalar.RelOp {
			parseRelOpRecursively(&relOp.ComputeScalar.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.StreamAggregate != nil {
		for i := range relOp.StreamAggregate.RelOp {
			parseRelOpRecursively(&relOp.StreamAggregate.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.Merge != nil {
		for i := range relOp.Merge.RelOp {
			parseRelOpRecursively(&relOp.Merge.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.Concatenation != nil {
		for i := range relOp.Concatenation.RelOp {
			parseRelOpRecursively(&relOp.Concatenation.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
	if relOp.Parallelism != nil {
		for i := range relOp.Parallelism.RelOp {
			parseRelOpRecursively(&relOp.Parallelism.RelOp[i], analysis, queryID, planHandle, nodeID, currentNodeID, "Input")
		}
	}
}

// checkForSpill checks if the operator experienced memory spills
func checkForSpill(relOp *RelOp) bool {
	// Look for spill indicators in sort, hash, and other memory-intensive operators
	opType := strings.ToLower(relOp.PhysicalOp)

	// Common operators that can spill
	spillOperators := []string{"sort", "hash match", "hash join", "stream aggregate", "window spool"}
	for _, spillOp := range spillOperators {
		if strings.Contains(opType, spillOp) {
			// In a real implementation, you'd check for specific spill warnings or memory grants
			// For now, we'll use a heuristic based on cost
			if cost, err := strconv.ParseFloat(relOp.EstimatedTotalSubtreeCost, 64); err == nil && cost > 10.0 {
				return true
			}
		}
	}
	return false
}

// checkForMissingJoinPredicate checks if a join operation is missing proper predicates
func checkForMissingJoinPredicate(relOp *RelOp) bool {
	opType := strings.ToLower(relOp.PhysicalOp)

	// Check for join operators
	joinOperators := []string{"nested loops", "hash match", "merge join"}
	for _, joinOp := range joinOperators {
		if strings.Contains(opType, joinOp) {
			// Check if estimated rows is suspiciously high (potential Cartesian product)
			if rows, err := strconv.ParseFloat(relOp.EstimateRows, 64); err == nil && rows > 1000000 {
				return true
			}
		}
	}
	return false
}

// ParseExecutionPlanTopLevelDetails parses execution plan XML to extract only top-level details
// This is much faster than full node parsing and suitable for high-volume active query monitoring
// Returns top-level plan information: costs, compilation details, optimizer info, and warnings
func ParseExecutionPlanTopLevelDetails(xmlContent string, queryID string, planHandle string, planHandleResult *PlanHandleResult) (*ExecutionPlanTopLevelDetails, error) {
	if xmlContent == "" {
		return nil, fmt.Errorf("empty execution plan XML")
	}

	// Parse XML
	var showPlan ShowPlanXML
	if err := xml.Unmarshal([]byte(xmlContent), &showPlan); err != nil {
		return nil, fmt.Errorf("failed to parse execution plan XML: %w", err)
	}

	// Initialize top-level details
	details := &ExecutionPlanTopLevelDetails{
		QueryID:             queryID,
		PlanHandle:          planHandle,
		CollectionTimestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Extract from PlanHandleResult (if provided)
	if planHandleResult != nil {
		if planHandleResult.ExecutionCount != nil {
			details.ExecutionCount = *planHandleResult.ExecutionCount
		}
		if planHandleResult.AvgElapsedTimeMs != nil {
			details.AvgElapsedTimeMs = *planHandleResult.AvgElapsedTimeMs
		}
		if planHandleResult.AvgWorkerTimeMs != nil {
			details.AvgWorkerTimeMs = *planHandleResult.AvgWorkerTimeMs
		}
		if planHandleResult.LastExecutionTime != nil {
			details.LastExecutionTime = *planHandleResult.LastExecutionTime
		}
		if planHandleResult.CreationTime != nil {
			details.CreationTime = *planHandleResult.CreationTime
		}
		if planHandleResult.QueryPlanHash != nil {
			details.QueryPlanHash = planHandleResult.QueryPlanHash.String()
		}
	}

	// Extract from first batch/statement (most execution plans have single batch/statement)
	if len(showPlan.BatchSequence) == 0 || len(showPlan.BatchSequence[0].Batch) == 0 || len(showPlan.BatchSequence[0].Batch[0].Statements) == 0 {
		return nil, fmt.Errorf("no statements found in execution plan")
	}

	stmt := showPlan.BatchSequence[0].Batch[0].Statements[0]

	// Statement details
	details.SQLText = stmt.StatementText
	if stmt.QueryPlanHash != "" && details.QueryPlanHash == "" {
		details.QueryPlanHash = stmt.QueryPlanHash
	}
	details.StatementOptmLevel = stmt.StatementOptmLevel

	// Statement costs
	if cost, err := strconv.ParseFloat(stmt.StatementSubTreeCost, 64); err == nil {
		details.StatementSubTreeCost = cost
		details.EstimatedTotalCost = cost
	}
	if rows, err := strconv.ParseFloat(stmt.StatementEstRows, 64); err == nil {
		details.EstimateRows = rows
	}

	// QueryPlan details
	plan := stmt.QueryPlan
	details.CompileTime = plan.CompileTime
	if cpu, err := strconv.ParseInt(plan.CompileCPU, 10, 64); err == nil {
		details.CompileCPU = cpu
	}
	if mem, err := strconv.ParseInt(plan.CompileMemory, 10, 64); err == nil {
		details.CompileMemory = mem
	}
	if size, err := strconv.ParseInt(plan.CachedPlanSize, 10, 64); err == nil {
		details.CachedPlanSize = size
		details.CachedPlanSize64 = size
	}
	if dop, err := strconv.ParseInt(plan.DegreeOfParallelism, 10, 64); err == nil {
		details.DegreeOfParallelism = dop
	}
	if mem, err := strconv.ParseInt(plan.MemoryGrant, 10, 64); err == nil {
		details.MemoryGrant = mem
	}

	// Root RelOp costs
	rootOp := plan.RelOp
	if cost, err := strconv.ParseFloat(rootOp.EstimatedTotalSubtreeCost, 64); err == nil {
		details.TotalSubtreeCost = cost
		if details.EstimatedTotalCost == 0 {
			details.EstimatedTotalCost = cost
		}
	}
	if io, err := strconv.ParseFloat(rootOp.EstimateIO, 64); err == nil {
		details.EstimateIO = io
	}
	if cpu, err := strconv.ParseFloat(rootOp.EstimateCPU, 64); err == nil {
		details.EstimateCPU = cpu
	}
	if size, err := strconv.ParseInt(rootOp.AvgRowSize, 10, 64); err == nil {
		details.AvgRowSize = size
	}

	// Analyze operators for summary statistics (lightweight scan)
	details.TotalOperators = 1 // Start with root
	details.ScansCount = 0
	details.SeeksCount = 0
	details.NoJoinPredicate = false
	details.ColumnsWithNoStatistics = 0
	details.UnmatchedIndexes = 0

	// Scan operators recursively (lightweight)
	scanOperatorsSummary(&rootOp, details)

	// Missing index warnings (placeholder - would need Warnings section parsing)
	details.MissingIndexCount = 0
	details.MissingIndexImpact = 0.0

	return details, nil
}

// scanOperatorsSummary performs lightweight recursive scan of operators for summary statistics
// Only counts operators and identifies scans/seeks (no detailed parsing)
func scanOperatorsSummary(relOp *RelOp, details *ExecutionPlanTopLevelDetails) {
	// Count this operator
	details.TotalOperators++

	// Check operator type
	opType := strings.ToLower(relOp.PhysicalOp)
	if strings.Contains(opType, "scan") {
		details.ScansCount++
	}
	if strings.Contains(opType, "seek") {
		details.SeeksCount++
	}
	if checkForMissingJoinPredicate(relOp) {
		details.NoJoinPredicate = true
	}

	// Recursively process child operators (lightweight)
	for i := range relOp.RelOp {
		scanOperatorsSummary(&relOp.RelOp[i], details)
	}

	// Process join children
	if relOp.OneSide != nil {
		scanOperatorsSummary(relOp.OneSide, details)
	}
	if relOp.OtherSide != nil {
		scanOperatorsSummary(relOp.OtherSide, details)
	}

	// Process operator-specific children (lightweight)
	if relOp.IndexScan != nil {
		for i := range relOp.IndexScan.RelOp {
			scanOperatorsSummary(&relOp.IndexScan.RelOp[i], details)
		}
	}
	if relOp.NestedLoops != nil {
		for i := range relOp.NestedLoops.RelOp {
			scanOperatorsSummary(&relOp.NestedLoops.RelOp[i], details)
		}
	}
	if relOp.Hash != nil {
		for i := range relOp.Hash.RelOp {
			scanOperatorsSummary(&relOp.Hash.RelOp[i], details)
		}
	}
	if relOp.Sort != nil {
		for i := range relOp.Sort.RelOp {
			scanOperatorsSummary(&relOp.Sort.RelOp[i], details)
		}
	}
	if relOp.Filter != nil {
		for i := range relOp.Filter.RelOp {
			scanOperatorsSummary(&relOp.Filter.RelOp[i], details)
		}
	}
	if relOp.ComputeScalar != nil {
		for i := range relOp.ComputeScalar.RelOp {
			scanOperatorsSummary(&relOp.ComputeScalar.RelOp[i], details)
		}
	}
	if relOp.StreamAggregate != nil {
		for i := range relOp.StreamAggregate.RelOp {
			scanOperatorsSummary(&relOp.StreamAggregate.RelOp[i], details)
		}
	}
	if relOp.Merge != nil {
		for i := range relOp.Merge.RelOp {
			scanOperatorsSummary(&relOp.Merge.RelOp[i], details)
		}
	}
	if relOp.Concatenation != nil {
		for i := range relOp.Concatenation.RelOp {
			scanOperatorsSummary(&relOp.Concatenation.RelOp[i], details)
		}
	}
	if relOp.Parallelism != nil {
		for i := range relOp.Parallelism.RelOp {
			scanOperatorsSummary(&relOp.Parallelism.RelOp[i], details)
		}
	}
}
