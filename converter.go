package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// JUnitTestSuites represents the root XML element
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite
type JUnitTestSuite struct {
	XMLName   xml.Name        `xml:"testsuite"`
	Name      string          `xml:"name,attr"`
	Tests     int             `xml:"tests,attr"`
	Failures  int             `xml:"failures,attr"`
	Errors    int             `xml:"errors,attr"`
	Time      float64         `xml:"time,attr"`
	Timestamp string          `xml:"timestamp,attr"`
	TestCases []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a test case
type JUnitTestCase struct {
	XMLName   xml.Name      `xml:"testcase"`
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	XMLName xml.Name `xml:"failure"`
	Message string   `xml:"message,attr"`
	Type    string   `xml:"type,attr"`
	Content string   `xml:",chardata"`
}

// JUnitSkipped represents a skipped test
type JUnitSkipped struct {
	XMLName xml.Name `xml:"skipped"`
	Message string   `xml:"message,attr,omitempty"`
}

// XCResultRoot represents the root of the XCResult JSON structure
type XCResultRoot struct {
	Devices   []Device   `json:"devices"`
	TestNodes []TestNode `json:"testNodes"`
}

// Device represents device information
type Device struct {
	Architecture string `json:"architecture"`
	DeviceID     string `json:"deviceId"`
	DeviceName   string `json:"deviceName"`
	ModelName    string `json:"modelName"`
	OsVersion    string `json:"osVersion"`
	Platform     string `json:"platform"`
}

// TestNode represents a node in the test hierarchy
type TestNode struct {
	Children          []TestNode        `json:"children,omitempty"`
	Name              string            `json:"name"`
	NodeType          string            `json:"nodeType"`
	Duration          string            `json:"duration"`
	Result            string            `json:"result"`
	NodeIdentifier    string            `json:"nodeIdentifier,omitempty"`
	SummaryRef        SummaryRef        `json:"summaryRef,omitempty"`
	ActivitySummaries ActivitySummaries `json:"activitySummaries,omitempty"`
}

// SummaryRef represents a reference to a summary
type SummaryRef struct {
	ID struct {
		Value string `json:"_value"`
	} `json:"id"`
}

// ActivitySummaries represents activity summaries
type ActivitySummaries struct {
	Values []ActivitySummaryEntry `json:"_values"`
}

// ActivitySummaryEntry represents an entry in activity summaries
type ActivitySummaryEntry struct {
	ActivitySummary ActivitySummary `json:"activitySummary"`
}

// ActivitySummary represents an activity summary
type ActivitySummary struct {
	Title    string `json:"title"`
	Messages []struct {
		StringValue string `json:"string_value"`
	} `json:"messages"`
}

// ConvertXCResultJSONToJUnitXML converts XCResult JSON to JUnit XML
func ConvertXCResultJSONToJUnitXML(jsonData []byte) ([]byte, error) {
	var root XCResultRoot
	if err := json.Unmarshal(jsonData, &root); err != nil {
		return nil, fmt.Errorf("failed to parse XCResult JSON: %w", err)
	}

	testSuites := JUnitTestSuites{
		TestSuites: []JUnitTestSuite{},
	}
	suiteMap := make(map[string]*JUnitTestSuite)

	processTestNodes(root.TestNodes, "", suiteMap)

	// Convert map to slice and calculate totals
	for _, suite := range suiteMap {
		suite.Tests = len(suite.TestCases)
		suite.Time = totalSuiteTime(suite.TestCases)
		testSuites.TestSuites = append(testSuites.TestSuites, *suite)
	}

	// Sort test suites and test cases
	sortTestSuites(&testSuites)

	// If no test suites were created, add a default one
	if len(testSuites.TestSuites) == 0 {
		testSuites.TestSuites = append(testSuites.TestSuites, JUnitTestSuite{
			Name:      "XCTest",
			Tests:     0,
			Failures:  0,
			Errors:    0,
			Time:      0,
			Timestamp: time.Now().Format(time.RFC3339),
		})
	}

	xmlData, err := xml.MarshalIndent(testSuites, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JUnit XML: %w", err)
	}

	return append([]byte(xml.Header), xmlData...), nil
}

func processTestNodes(nodes []TestNode, classname string, suiteMap map[string]*JUnitTestSuite) {
	for _, node := range nodes {
		switch node.NodeType {
		case "Unit test bundle", "UI test bundle", "Test Suite":
			newClassname := buildClassName(classname, node.Name)
			processTestNodes(node.Children, newClassname, suiteMap)

		case "Test Case":
			processTestCase(node, classname, suiteMap)

		case "Test Plan", "Test Plan Configuration":
			// Process children of Test Plan nodes
			processTestNodes(node.Children, classname, suiteMap)

		case "Failure Message":
			// Handled in test case processing
		}
	}
}

func processTestCase(node TestNode, classname string, suiteMap map[string]*JUnitTestSuite) {
	// Skip test configurations, only process actual test cases
	if !strings.Contains(node.NodeIdentifier, "/") {
		return
	}

	parts := strings.Split(node.NodeIdentifier, "/")
	if len(parts) < 2 {
		return
	}

	suiteName := parts[0]
	if suiteName == "" {
		suiteName = "UnknownSuite"
	}

	// Get or create test suite
	suite, exists := suiteMap[suiteName]
	if !exists {
		suite = &JUnitTestSuite{
			Name:      suiteName,
			Timestamp: time.Now().Format(time.RFC3339),
			TestCases: []JUnitTestCase{},
		}
		suiteMap[suiteName] = suite
	}

	// Parse duration
	duration := parseDuration(node.Duration)

	// Create test case
	testCase := JUnitTestCase{
		Name:      node.Name,
		Classname: classname,
		Time:      duration,
	}

	// Handle failures
	if node.Result == "Failed" {
		failureMessage := extractFailureMessage(node)
		testCase.Failure = &JUnitFailure{
			Message: failureMessage,
			Type:    "Failure",
			Content: failureMessage,
		}
		suite.Failures++
	}

	suite.TestCases = append(suite.TestCases, testCase)
}

func parseDuration(dur string) float64 {
	dur = strings.TrimSuffix(dur, "s")
	if dur == "" {
		return 0
	}
	seconds, _ := strconv.ParseFloat(dur, 64)
	return seconds
}

func extractFailureMessage(node TestNode) string {
	for _, child := range node.Children {
		if child.NodeType == "Failure Message" {
			return child.Name
		}

		// Check deeper children
		message := extractFailureMessage(child)
		if message != "Test failed" {
			return message
		}
	}
	return "Test failed"
}

func buildClassName(current, newPart string) string {
	if current == "" {
		return newPart
	}
	return current + "." + newPart
}

func totalSuiteTime(cases []JUnitTestCase) float64 {
	var total float64
	for _, tc := range cases {
		total += tc.Time
	}
	return total
}

func sortTestSuites(suites *JUnitTestSuites) {
	// Sort test suites
	sort.Slice(suites.TestSuites, func(i, j int) bool {
		return suites.TestSuites[i].Name < suites.TestSuites[j].Name
	})

	// Sort test cases within each suite
	for i := range suites.TestSuites {
		sort.Slice(suites.TestSuites[i].TestCases, func(a, b int) bool {
			return suites.TestSuites[i].TestCases[a].Name < suites.TestSuites[i].TestCases[b].Name
		})
	}
}
