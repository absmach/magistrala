package main

import (
	"encoding/json"
	"fmt"

	"github.com/absmach/magistrala/auth"
)

func main() {
	var pat auth.PAT

	jsonData := `
	{
		"id": "id_1",
		"user": "user_1",
		"name": "user 1 PAT",
		"Description": "user 1 pat 1 description",
		"Token": "hashed token",
		"scope": {
			"users": {
				"operations": {
					"update": [
						"123",
						"123123"
					],
					"create": "123",
					"read": "*"
				}
			},
			"domains": {
				"domain_1": {
					"domain_management": {
						"operations": {
							"update": [
								"123",
								"123123"
							],
							"create": "123",
							"read": "*"
						}
					},
					"entities": {
						"groups": {
							"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
						},
						"things": {
							"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
						},
						"channels": {
							"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
						}
					}
				}
			}
		},
		"issued_at": "2024-01-01T00:00:00Z",
		"expires_at": "2024-01-04T00:00:00Z",
		"updated_at": "2024-01-02T00:00:00Z",
		"last_used_at": "2024-01-02T00:00:00Z",
		"revoked": true,
		"revoked_at": "2024-01-03T00:00:00Z"
	}
	`

	if err := json.Unmarshal([]byte(jsonData), &pat); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("pat: %s\n", pat.String())

	scopeByte := `
		{
					"domain_management": {
						"operations": {
							"update": [
								"123",
								"123123"
							],
							"create": "123",
							"read": "*"
						}
					},
					"entities": {
						"groups": {
							"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
						},
						"things": {
							"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
						},
						"channels": {
							"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
						}
					}
}
	`
	var domainscope auth.DomainScope

	if err := json.Unmarshal([]byte(scopeByte), &domainscope); err != nil {
		panic(err)
	}

	domBytes, _ := json.MarshalIndent(domainscope, "", "  ")
	fmt.Println("\n\n", string(domBytes))

	operationsByte := `
	{
		"operations": {
								"update": [
									"123",
									"123123"
								],
								"create": "123",
								"read": "*"
							}
	}

	`

	var operation auth.OperationScope

	if err := json.Unmarshal([]byte(operationsByte), &operation); err != nil {
		panic(err)
	}

	opbytes, _ := json.MarshalIndent(operation, "", "  ")
	fmt.Println("\n\n", string(opbytes))

}
