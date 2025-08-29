package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
)

type CustomerSegmentMember struct {
	Node Node `json:"node"`
}

type Node struct {
	ID                  string         `json:"id"`
	DisplayName         string         `json:"displayName"`
	DefaultEmailAddress *DefaultEmail  `json:"defaultEmailAddress,omitempty"`
	AmountSpent         MonetaryAmount `json:"amountSpent"`
}

type DefaultEmail struct {
	EmailAddress string `json:"emailAddress"`
}

type MonetaryAmount struct {
	Amount       decimal.Decimal `json:"amount"`
	CurrencyCode string          `json:"currencyCode"`
}

type GraphQLResponse struct {
	Data struct {
		CustomerSegmentMembers struct {
			Edges []CustomerSegmentMember `json:"edges"`
		} `json:"customerSegmentMembers"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type GraphQLError struct {
	Message string `json:"message"`
}

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func main() {
	// Load environment variables locally
	_ = godotenv.Load()

	app := &cli.App{
		Name:  "shopify-customers",
		Usage: "Fetch Shopify customer segment members and export to CSV",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "query", Value: "customer_tags CONTAINS 'task1' AND customer_tags CONTAINS 'level:3'", Aliases: []string{"q"}, Usage: "GraphQL query string"},
			&cli.IntFlag{Name: "first", Value: 50, Aliases: []string{"f"}, Usage: "Number of customers to fetch"},
			&cli.StringFlag{Name: "sortKey", Value: "amount_spent", Aliases: []string{"s"}, Usage: "Sort key for results"},
			&cli.BoolFlag{Name: "reverse", Value: true, Aliases: []string{"r"}, Usage: "Reverse sort order"},
			&cli.StringFlag{Name: "output", Value: "customers.csv", Aliases: []string{"o"}, Usage: "Output CSV filename (leave empty for stdout)"},
		},
		Action: func(c *cli.Context) error {
			// Set a global 5-second timeout
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return fetchAndExportCustomers(ctx, c)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func fetchAndExportCustomers(ctx context.Context, c *cli.Context) error {
	shopifyDomain := os.Getenv("SHOPIFY_DOMAIN")
	accessToken := os.Getenv("SHOPIFY_ACCESS_TOKEN")
	if shopifyDomain == "" || accessToken == "" {
		return fmt.Errorf("SHOPIFY_DOMAIN and SHOPIFY_ACCESS_TOKEN must be set")
	}

	variables := map[string]interface{}{
		"first":   c.Int("first"),
		"query":   c.String("query"),
		"sortKey": c.String("sortKey"),
		"reverse": c.Bool("reverse"),
	}

	query := `
	query GetCustomerSegmentMembers($first: Int!, $query: String!, $sortKey: String, $reverse: Boolean!) {
		customerSegmentMembers(first: $first, query: $query, sortKey: $sortKey, reverse: $reverse) {
			edges {
				node {
					id
					displayName
					defaultEmailAddress {
						emailAddress
					}
					amountSpent {
						amount
						currencyCode
					}
				}
			}
		}
	}`

	resp, err := executeGraphQLQuery(ctx, shopifyDomain, accessToken, GraphQLRequest{
		Query:     query,
		Variables: variables,
	})
	if err != nil {
		return fmt.Errorf("GraphQL query failed: %w", err)
	}

	if len(resp.Errors) > 0 {
		return fmt.Errorf("GraphQL errors: %v", resp.Errors)
	}

	if err := exportToCSV(ctx, resp.Data.CustomerSegmentMembers.Edges, c.String("output")); err != nil {
		return fmt.Errorf("failed to export CSV: %w", err)
	}

	fmt.Printf("Successfully exported %d customers to %s\n", len(resp.Data.CustomerSegmentMembers.Edges), c.String("output"))
	return nil
}

func executeGraphQLQuery(ctx context.Context, domain, accessToken string, request GraphQLRequest) (*GraphQLResponse, error) {
	url := fmt.Sprintf("https://%s/admin/api/2025-01/graphql.json", domain)

	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Shopify-Access-Token", accessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("operation timed out after 5 seconds")
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(b))
	}

	var graphqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &graphqlResp, nil
}

func exportToCSV(ctx context.Context, customers []CustomerSegmentMember, filename string) error {
	var writer *csv.Writer
	var file *os.File
	var err error

	if filename == "" {
		writer = csv.NewWriter(os.Stdout)
	} else {
		file, err = os.Create(filename)
		if err != nil {
			return err
		}
		defer file.Close()
		writer = csv.NewWriter(file)
	}
	defer writer.Flush()

	header := []string{"ID", "Display Name", "Email Address", "Amount Spent", "Currency Code"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, c := range customers {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation timed out during CSV export")
		default:
			email := ""
			if c.Node.DefaultEmailAddress != nil {
				email = c.Node.DefaultEmailAddress.EmailAddress
			}
			record := []string{
				c.Node.ID,
				c.Node.DisplayName,
				email,
				c.Node.AmountSpent.Amount.StringFixed(2),
				c.Node.AmountSpent.CurrencyCode,
			}
			if err := writer.Write(record); err != nil {
				return err
			}
		}
	}
	return nil
}
