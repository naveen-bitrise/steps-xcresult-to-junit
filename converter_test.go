package main

import (
	"encoding/json"
	"testing"
)

func TestProcessXCResultJSON(t *testing.T) {
	// Create a minimal valid XCResult JSON structure
	minimalXCResult := map[string]interface{}{
		"testPlanSummaries": map[string]interface{}{
			"summaries": []interface{}{
				map[string]interface{}{
					"testableSummaries": map[string]interface{}{
						"_values": []interface{}{
							map[string]interface{}{
								"name": map[string]interface{}{
									"_value": "MyTestSuite",
								},
								"tests": map[string]interface{}{
									"_values": []interface{}{
										map[string]interface{}{
											"name": map[string]interface{}{
												"_value": "MyTestGroup",
											},
											"subtests": map[string]interface{}{
												"_values": []interface{}{
													map[string]interface{}{
														"name": map[string]interface{}{
															"_value": "testExample",
														},
														"duration":   0.5,
														"testStatus": "Success",
													},
												},
											},
										},
									},
								},
								"duration":     1.5,
								"testCount":    1,
								"failureCount": 0,
								"skipCount":    0,
							},
						},
					},
				},
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(minimalXCResult)
	if err != nil {
		t.Fatalf("Failed to marshal test JSON: %v", err)
	}

	// Process the JSON
	testSuites, err := processXCResultJSON(jsonData)
	if err != nil {
		t.Fatalf("processXCResultJSON returned error: %v", err)
	}

	// Validate the results
	if len(testSuites.TestSuites) != 1 {
		t.Fatalf("Expected 1 test suite, got %d", len(testSuites.TestSuites))
	}

	suite := testSuites.TestSuites[0]
	if suite.Name != "MyTestSuite" {
		t.Errorf("Expected suite name to be MyTestSuite, got %s", suite.Name)
	}
	if suite.Tests != 1 {
		t.Errorf("Expected 1 test, got %d", suite.Tests)
	}
	if suite.Failures != 0 {
		t.Errorf("Expected 0 failures, got %d", suite.Failures)
	}
	if suite.Time != 1.5 {
		t.Errorf("Expected duration 1.5, got %f", suite.Time)
	}

	// Now test with a failing test case
	minimalXCResult["testPlanSummaries"].(map[string]interface{})["summaries"].([]interface{})[0].(map[string]interface{})["testableSummaries"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["tests"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["subtests"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["testStatus"] = "Failure"
	minimalXCResult["testPlanSummaries"].(map[string]interface{})["summaries"].([]interface{})[0].(map[string]interface{})["testableSummaries"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["tests"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["subtests"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["failureSummaries"] = map[string]interface{}{
		"_values": []interface{}{
			map[string]interface{}{
				"message": map[string]interface{}{
					"_value": "Test failed: expected true but got false",
				},
			},
		},
	}
	minimalXCResult["testPlanSummaries"].(map[string]interface{})["summaries"].([]interface{})[0].(map[string]interface{})["testableSummaries"].(map[string]interface{})["_values"].([]interface{})[0].(map[string]interface{})["failureCount"] = 1

	// Convert to JSON
	jsonData, err = json.Marshal(minimalXCResult)
	if err != nil {
		t.Fatalf("Failed to marshal test JSON: %v", err)
	}

	// Process the JSON
	testSuites, err = processXCResultJSON(jsonData)
	if err != nil {
		t.Fatalf("processXCResultJSON returned error: %v", err)
	}

	// Validate the results
	if len(testSuites.TestSuites) != 1 {
		t.Fatalf("Expected 1 test suite, got %d", len(testSuites.TestSuites))
	}

	suite = testSuites.TestSuites[0]
	if suite.Failures != 1 {
		t.Errorf("Expected 1 failure, got %d", suite.Failures)
	}

	if len(suite.TestCases) != 1 {
		t.Fatalf("Expected 1 test case, got %d", len(suite.TestCases))
	}

	testCase := suite.TestCases[0]
	if testCase.Failure == nil {
		t.Errorf("Expected failure to be set, got nil")
	} else if testCase.Failure.Message != "Test failed: expected true but got false" {
		t.Errorf("Expected failure message 'Test failed: expected true but got false', got '%s'", testCase.Failure.Message)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test map for helper functions
	testMap := map[string]interface{}{
		"string": "value",
		"number": 42.5,
		"nested": map[string]interface{}{
			"string": "nested value",
			"number": 99.9,
		},
	}

	// Test getValueByPath
	t.Run("getValueByPath", func(t *testing.T) {
		value := getValueByPath(testMap, []string{"string"})
		if value != "value" {
			t.Errorf("Expected 'value', got %v", value)
		}

		value = getValueByPath(testMap, []string{"nested", "string"})
		if value != "nested value" {
			t.Errorf("Expected 'nested value', got %v", value)
		}

		value = getValueByPath(testMap, []string{"nonexistent"})
		if value != nil {
			t.Errorf("Expected nil, got %v", value)
		}
	})

	// Test getStringByPath
	t.Run("getStringByPath", func(t *testing.T) {
		value := getStringByPath(testMap, []string{"string"})
		if value != "value" {
			t.Errorf("Expected 'value', got %v", value)
		}

		value = getStringByPath(testMap, []string{"number"})
		if value != "" {
			t.Errorf("Expected empty string, got %v", value)
		}
	})

	// Test getFloatByPath
	t.Run("getFloatByPath", func(t *testing.T) {
		value := getFloatByPath(testMap, []string{"number"})
		if value != 42.5 {
			t.Errorf("Expected 42.5, got %v", value)
		}

		value = getFloatByPath(testMap, []string{"string"})
		if value != 0 {
			t.Errorf("Expected 0, got %v", value)
		}
	})

	// Test getIntByPath
	t.Run("getIntByPath", func(t *testing.T) {
		value := getIntByPath(testMap, []string{"number"})
		if value != 42 {
			t.Errorf("Expected 42, got %v", value)
		}

		value = getIntByPath(testMap, []string{"string"})
		if value != 0 {
			t.Errorf("Expected 0, got %v", value)
		}
	})
}
