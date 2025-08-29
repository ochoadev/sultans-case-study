# Shopify Customer Segment Exporter

A Go application that fetches Shopify customer segment members using the Admin GraphQL API and exports them to CSV format.

## Prerequisites

- Go 1.19 or higher installed on your system
- A Shopify store with Admin API access
- Shopify access token with appropriate permissions

## Installation

1. Clone or download this repository
2. Navigate to the project directory
3. Install dependencies:
   ```bash
   go mod download
   ```

## Configuration

Create a `.env` file in the project root with your Shopify credentials:

```env
SHOPIFY_DOMAIN=your-store.myshopify.com
SHOPIFY_ACCESS_TOKEN=your_access_token_here
```

**Note:** Replace `your-store.myshopify.com` with your actual Shopify domain and `your_access_token_here` with your Shopify Admin API access token.

## Usage

### Basic Usage

Run the application with default settings:

```bash
go run main.go
```

This will:
- Fetch up to 50 customers with tags containing 'task1' AND 'level:3'
- Sort by amount spent in descending order
- Export results to `customers.csv`

### Command Line Options

The application supports several command-line flags:

```bash
go run main.go [OPTIONS]
```

#### Available Flags:

- `--query, -q`: GraphQL query string (default: `"customer_tags CONTAINS 'task1' AND customer_tags CONTAINS 'level:3'"`)
- `--first, -f`: Number of customers to fetch (default: 50)
- `--sortKey, -s`: Sort key for results (default: "amount_spent")
- `--reverse, -r`: Reverse sort order (default: true)
- `--output, -o`: Output CSV filename (default: "customers.csv", use empty string for stdout)

### Examples

#### Fetch 100 customers with custom query:
```bash
go run main.go --first 100 --query "customer_tags CONTAINS 'vip'"
```

#### Export to custom filename:
```bash
go run main.go --output "vip_customers.csv"
```

#### Output to console instead of file:
```bash
go run main.go --output ""
```

#### Custom sorting:
```bash
go run main.go --sortKey "created_at" --reverse false
```

## Building

To create an executable binary:

```bash
go build -o shopify-customers main.go
```

Then run the binary:

```bash
./shopify-customers
```

## Output Format

The CSV output contains the following columns:
- **ID**: Shopify customer ID
- **Display Name**: Customer's display name
- **Email Address**: Customer's email address
- **Amount Spent**: Total amount spent by the customer
- **Currency Code**: Currency of the amount spent

## Error Handling

- The application has a 5-second timeout for all operations
- Missing environment variables will result in an error
- GraphQL errors are displayed with details
- HTTP errors include status codes and response bodies

## Dependencies

- `github.com/joho/godotenv`: Environment variable loading
- `github.com/shopify/spring/decimal`: Decimal number handling
- `github.com/urfave/cli/v2`: Command-line interface

## Troubleshooting

### Common Issues:

1. **"SHOPIFY_DOMAIN and SHOPIFY_ACCESS_TOKEN must be set"**
   - Ensure your `.env` file exists and contains the required variables
   - Check that the variable names are exactly as shown

2. **"operation timed out after 5 seconds"**
   - The request is taking too long, consider reducing the `--first` parameter
   - Check your network connection and Shopify API status

3. **"GraphQL errors"**
   - Verify your access token has the necessary permissions
   - Check that your query syntax is valid for the Shopify Admin API

4. **Permission denied errors**
   - Ensure your Shopify access token has the required scopes for customer data

## License

This project is open source and available under the MIT License.
