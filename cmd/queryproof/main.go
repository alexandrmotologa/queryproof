package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/alexandrmotologa/queryproof/internal/config"
	"github.com/alexandrmotologa/queryproof/internal/dbt"
	"github.com/alexandrmotologa/queryproof/internal/engine"
	"github.com/alexandrmotologa/queryproof/internal/introspect"
	"github.com/alexandrmotologa/queryproof/internal/ledger"
	"github.com/alexandrmotologa/queryproof/internal/metrics"
	"github.com/alexandrmotologa/queryproof/internal/probes"
	"github.com/alexandrmotologa/queryproof/internal/proof"
	"github.com/alexandrmotologa/queryproof/internal/proxy"
	"github.com/alexandrmotologa/queryproof/internal/server"
	"github.com/alexandrmotologa/queryproof/web"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcmd := os.Args[1]
	switch subcmd {
	case "verify":
		runVerify(os.Args[2:])
	case "rewrite":
		runRewrite(os.Args[2:])
	case "audit":
		runAudit(os.Args[2:])
	case "passport":
		runPassport(os.Args[2:])
	case "extract":
		runExtract(os.Args[2:])
	case "import-dbt":
		runImportDbt(os.Args[2:])
	case "proxy":
		runProxy(os.Args[2:])
	case "serve":
		runServe(os.Args[2:])
	case "version":
		printVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand %q. Run 'queryproof help' for usage.\n", subcmd)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`queryproof - Deterministic Text-to-SQL Verification & Falsification Engine (v%s)

Usage:
  queryproof <command> [arguments]

Commands:
  verify      Verify candidate SQL against probes, rewrite AST, and issue proof.json
  rewrite     Output only the certified, rewritten SQL to stdout (pipeline-friendly)
  audit       Run falsification audit on a SQL query or directory of queries
  passport    Cryptographically verify the integrity and signature of a proof.json file
  extract     Introspect live database schema into catalog.json automatically
  import-dbt  Import dbt project models, tests, and metrics into catalog and metrics.yaml
  proxy       Run standalone agent gateway interceptor proxy
  serve       Launch embedded React 19 BI Auditor Cockpit & REST API server
  version     Display engine version and memory consumption statistics

Flags for 'verify', 'rewrite', 'audit':
  -q, --query    Raw SQL query string
  -f, --file     Path to .sql file
  -c, --catalog  Path to catalog schema (JSON or YAML) [default: examples/catalog.json]
  -m, --metrics  Path to metrics.yaml [default: examples/metrics.yaml]
  -d, --dialect  Target SQL dialect (postgres, snowflake, bigquery) [default: postgres]
  -o, --out      Output path for signed proof voucher (e.g. proof.json)
      --json     Output full verification report in JSON format

Run 'queryproof <command> --help' for details on a specific command.
`, version)
}

func printVersion() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("queryproof version %s\n", version)
	fmt.Printf("Go runtime: %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Memory footprint: %.2f MB\n", float64(m.Alloc)/(1024*1024))
}

func resolveCatalogAndMetrics(catPath, metPath string) (*config.Catalog, *metrics.Registry) {
	if catPath == "" {
		candidates := []string{"catalog.json", "examples/catalog.json", "../examples/catalog.json"}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				catPath = c
				break
			}
		}
	}
	if metPath == "" {
		candidates := []string{"metrics.yaml", "examples/metrics.yaml", "../examples/metrics.yaml"}
		for _, m := range candidates {
			if _, err := os.Stat(m); err == nil {
				metPath = m
				break
			}
		}
	}

	var cat *config.Catalog
	if catPath != "" {
		if c, err := config.LoadCatalog(catPath); err == nil {
			cat = c
		}
	}
	if cat == nil {
		cat = &config.Catalog{Dialect: config.DialectPostgres}
		cat.Init()
	}

	var reg *metrics.Registry
	if metPath != "" {
		if r, err := metrics.LoadMetrics(metPath); err == nil {
			reg = r
		}
	}

	return cat, reg
}

func runVerify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	queryStr := fs.String("query", "", "Candidate SQL query")
	fs.StringVar(queryStr, "q", "", "Candidate SQL query (shorthand)")
	filePath := fs.String("file", "", "Path to .sql file")
	fs.StringVar(filePath, "f", "", "Path to .sql file (shorthand)")
	catPath := fs.String("catalog", "", "Catalog path")
	fs.StringVar(catPath, "c", "", "Catalog path (shorthand)")
	metPath := fs.String("metrics", "", "Metrics path")
	fs.StringVar(metPath, "m", "", "Metrics path (shorthand)")
	dialectStr := fs.String("dialect", "postgres", "Dialect")
	fs.StringVar(dialectStr, "d", "postgres", "Dialect (shorthand)")
	outPath := fs.String("out", "", "Proof voucher output path")
	fs.StringVar(outPath, "o", "", "Proof voucher output path (shorthand)")
	asJSON := fs.Bool("json", false, "Output full report in JSON")
	_ = fs.Parse(args)

	sql := *queryStr
	if sql == "" && *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading query file: %v\n", err)
			os.Exit(1)
		}
		sql = string(data)
	}
	if sql == "" {
		fmt.Fprintln(os.Stderr, "Error: please supply a query via -q/--query or -f/--file")
		os.Exit(1)
	}

	cat, reg := resolveCatalogAndMetrics(*catPath, *metPath)
	cfg := config.DefaultConfig()
	cfg.Dialect = config.Dialect(*dialectStr)

	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize engine: %v\n", err)
		os.Exit(1)
	}

	report, err := eng.Verify(sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification failed with error: %v\n", err)
		os.Exit(1)
	}

	if *outPath != "" && report.Voucher != nil {
		if err := report.Voucher.SaveToFile(*outPath); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save proof voucher to %s: %v\n", *outPath, err)
		} else {
			fmt.Printf("Cryptographic proof voucher written to %s\n", *outPath)
		}
	}

	if *asJSON {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		return
	}

	printReportSummary(report)
}

func runRewrite(args []string) {
	fs := flag.NewFlagSet("rewrite", flag.ExitOnError)
	queryStr := fs.String("query", "", "Candidate SQL query")
	fs.StringVar(queryStr, "q", "", "Candidate SQL query (shorthand)")
	filePath := fs.String("file", "", "Path to .sql file")
	fs.StringVar(filePath, "f", "", "Path to .sql file (shorthand)")
	catPath := fs.String("catalog", "", "Catalog path")
	fs.StringVar(catPath, "c", "", "Catalog path (shorthand)")
	metPath := fs.String("metrics", "", "Metrics path")
	fs.StringVar(metPath, "m", "", "Metrics path (shorthand)")
	dialectStr := fs.String("dialect", "postgres", "Dialect")
	fs.StringVar(dialectStr, "d", "postgres", "Dialect (shorthand)")
	_ = fs.Parse(args)

	sql := *queryStr
	if sql == "" && *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading query file: %v\n", err)
			os.Exit(1)
		}
		sql = string(data)
	}
	if sql == "" {
		fmt.Fprintln(os.Stderr, "Error: please supply a query via -q/--query or -f/--file")
		os.Exit(1)
	}

	cat, reg := resolveCatalogAndMetrics(*catPath, *metPath)
	cfg := config.DefaultConfig()
	cfg.Dialect = config.Dialect(*dialectStr)

	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine error: %v\n", err)
		os.Exit(1)
	}

	report, err := eng.Verify(sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if report.IsClean {
		fmt.Println(report.OriginalSQL)
	} else if report.RewrittenSQL != "" {
		fmt.Println(report.RewrittenSQL)
	} else {
		fmt.Println(report.OriginalSQL)
	}
}

func runAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to .sql file or directory")
	fs.StringVar(filePath, "f", "", "Path to .sql file or directory (shorthand)")
	catPath := fs.String("catalog", "", "Catalog path")
	fs.StringVar(catPath, "c", "", "Catalog path (shorthand)")
	metPath := fs.String("metrics", "", "Metrics path")
	fs.StringVar(metPath, "m", "", "Metrics path (shorthand)")
	dialectStr := fs.String("dialect", "postgres", "Dialect")
	fs.StringVar(dialectStr, "d", "postgres", "Dialect (shorthand)")
	_ = fs.Parse(args)

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "Error: please specify a file or directory via -f/--file")
		os.Exit(1)
	}

	cat, reg := resolveCatalogAndMetrics(*catPath, *metPath)
	cfg := config.DefaultConfig()
	cfg.Dialect = config.Dialect(*dialectStr)

	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine init error: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(*filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot access path %s: %v\n", *filePath, err)
		os.Exit(1)
	}

	var files []string
	if info.IsDir() {
		entries, _ := os.ReadDir(*filePath)
		for _, e := range entries {
			if strings.HasSuffix(strings.ToLower(e.Name()), ".sql") {
				files = append(files, filepath.Join(*filePath, e.Name()))
			}
		}
	} else {
		files = append(files, *filePath)
	}

	fmt.Printf("\n=== queryproof Falsification Audit Report (%d files) ===\n\n", len(files))

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("File: %s [READ ERROR: %v]\n", filepath.Base(file), err)
			continue
		}

		rep, err := eng.Verify(string(data))
		if err != nil {
			fmt.Printf("File: %s [PARSE ERROR: %v]\n", filepath.Base(file), err)
			continue
		}

		statusLabel := "CERTIFIED (CLEAN)"
		if !rep.IsClean {
			statusLabel = fmt.Sprintf("FALSIFIED (%d VIOLATIONS)", len(rep.SuiteResult.AllViolations))
		}

		fmt.Printf("File: %s | Status: %s | Score: %.0f%% | Latency: %.2fms\n",
			filepath.Base(file), statusLabel, rep.SuiteResult.OverallScore*100, rep.VerificationMs)

		for _, v := range rep.SuiteResult.AllViolations {
			fmt.Printf("   [%s] %s: %s\n", v.Severity, v.Title, v.Message)
		}
		if rep.RewrittenSQL != "" {
			fmt.Printf("   -> Auto-Rewritten via %d AST mutations\n", len(rep.Mutations))
		}
		fmt.Println()
	}
}

func runPassport(args []string) {
	fs := flag.NewFlagSet("passport", flag.ExitOnError)
	proofPath := fs.String("proof", "proof.json", "Path to proof.json voucher file")
	secretKey := fs.String("key", "queryproof-enterprise-master-key", "HMAC verification secret key")
	_ = fs.Parse(args)

	voucher, err := proof.LoadFromFile(*proofPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load proof voucher from %s: %v\n", *proofPath, err)
		os.Exit(1)
	}

	valid, err := proof.VerifyVoucher(voucher, *secretKey)
	if err != nil || !valid {
		fmt.Fprintf(os.Stderr, "INVALID PASSPORT: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("PASSPORT VERIFIED VALID")
	fmt.Printf("Voucher ID:    %s\n", voucher.VoucherID)
	fmt.Printf("Status:        %s\n", voucher.Status)
	fmt.Printf("Engine:        %s\n", voucher.Engine)
	fmt.Printf("Issued At:     %s\n", voucher.IssuedAt)
	fmt.Printf("Integrity:     %.2f%%\n", voucher.IntegrityScore*100)
	fmt.Printf("Signature Key: %s\n", voucher.Signature.KeyID)
	fmt.Printf("HMAC Digest:   %s\n", voucher.Signature.Digest)
}

func runExtract(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	driver := fs.String("driver", "sqlite", "Database driver (sqlite, postgres)")
	dsn := fs.String("dsn", "", "Database DSN / connection string")
	outPath := fs.String("out", "catalog.json", "Output path for extracted catalog.json")
	_ = fs.Parse(args)

	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "Error: --dsn is required for schema extraction")
		os.Exit(1)
	}

	cat, err := introspect.ExtractFromDatabase(context.Background(), *driver, *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Extraction failed: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal catalog: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outPath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write catalog to %s: %v\n", *outPath, err)
		os.Exit(1)
	}

	fmt.Printf("Extracted %d tables and %d relationships into %s\n", len(cat.Tables), len(cat.Relationships), *outPath)
}

func runImportDbt(args []string) {
	fs := flag.NewFlagSet("import-dbt", flag.ExitOnError)
	dirPath := fs.String("dir", ".", "Path to dbt project models directory")
	outCat := fs.String("out-catalog", "catalog.json", "Output catalog.json path")
	outMet := fs.String("out-metrics", "metrics.yaml", "Output metrics.yaml path")
	_ = fs.Parse(args)

	cat, dict, err := dbt.ImportDbtProject(*dirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dbt import failed: %v\n", err)
		os.Exit(1)
	}

	catData, _ := json.MarshalIndent(cat, "", "  ")
	_ = os.WriteFile(*outCat, catData, 0644)

	metData, _ := yaml.Marshal(dict)
	_ = os.WriteFile(*outMet, metData, 0644)

	fmt.Printf("Imported dbt project: %d tables, %d relationships into %s, %d metrics into %s\n",
		len(cat.Tables), len(cat.Relationships), *outCat, len(dict.Metrics), *outMet)
}

func runProxy(args []string) {
	fs := flag.NewFlagSet("proxy", flag.ExitOnError)
	port := fs.Int("port", 8080, "Agent gateway proxy port")
	catPath := fs.String("catalog", "", "Catalog path")
	metPath := fs.String("metrics", "", "Metrics path")
	_ = fs.Parse(args)

	cat, reg := resolveCatalogAndMetrics(*catPath, *metPath)
	cfg := config.DefaultConfig()

	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine error: %v\n", err)
		os.Exit(1)
	}

	led, _ := ledger.OpenLedger(".queryproof/audit.db")
	gw := proxy.NewAgentGateway(eng, led)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/query/intercept", gw.HandleIntercept)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"proxy-healthy"}`))
	})

	addr := fmt.Sprintf(":%d", *port)
	fmt.Printf("[queryproof] Text-to-SQL Agent Gateway Proxy running on http://127.0.0.1%s\n", addr)
	fmt.Printf("Intercept endpoint: POST http://127.0.0.1%s/v1/query/intercept\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "Proxy failed: %v\n", err)
		os.Exit(1)
	}
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 7373, "Server port")
	host := fs.String("host", "127.0.0.1", "Server host")
	catPath := fs.String("catalog", "", "Catalog path")
	metPath := fs.String("metrics", "", "Metrics path")
	_ = fs.Parse(args)

	cat, reg := resolveCatalogAndMetrics(*catPath, *metPath)
	cfg := config.DefaultConfig()
	cfg.Server.Port = *port
	cfg.Server.Host = *host

	eng, err := engine.NewEngine(cfg, cat, reg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Engine init error: %v\n", err)
		os.Exit(1)
	}

	led, _ := ledger.OpenLedger(".queryproof/audit.db")
	gw := proxy.NewAgentGateway(eng, led)
	srv := server.NewServer(&cfg.Server, eng, cat, reg, led, gw, cfg.SecretKey, web.Assets())

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf(`
╔═══════════════════════════════════════════════════════════════════╗
║                      queryproof v%s                            ║
║       Autonomous Text-to-SQL Verification Engine & Studio         ║
╠═══════════════════════════════════════════════════════════════════╣
║  Cockpit Studio:  http://%s:%d                             ║
║  REST Endpoints:  /api/verify, /api/catalog, /api/metrics         ║
║  Target Dialects: PostgreSQL, Snowflake, BigQuery                 ║
║  Memory Target:   < 20MB RAM Footprint                            ║
╚═══════════════════════════════════════════════════════════════════╝
Press Ctrl+C to terminate server.
`, version, *host, *port)

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
	fmt.Println("\nServer gracefully stopped.")
}

func printReportSummary(rep *engine.VerificationReport) {
	fmt.Println("\n-----------------------------------------------------------")
	fmt.Printf("queryproof Verification Verdict: ")
	if rep.IsClean {
		fmt.Printf("CERTIFIED (Score: %.0f%%)\n", rep.SuiteResult.OverallScore*100)
	} else {
		fmt.Printf("FALSIFIED & REWRITTEN (Score: %.0f%%)\n", rep.SuiteResult.OverallScore*100)
	}
	fmt.Printf("Verification Latency: %.2f ms\n", rep.VerificationMs)
	fmt.Printf("Candidate Hash:       %s\n", rep.OriginalHash)

	if rep.RewrittenHash != "" {
		fmt.Printf("Rewritten Hash:       %s\n", rep.RewrittenHash)
	}
	fmt.Println("-----------------------------------------------------------")

	if len(rep.SuiteResult.AllViolations) > 0 {
		fmt.Printf("\nViolations Detected (%d):\n", len(rep.SuiteResult.AllViolations))
		for i, v := range rep.SuiteResult.AllViolations {
			fmt.Printf("%d. [%s] %s\n", i+1, v.Severity, v.Title)
			fmt.Printf("   Message:     %s\n", v.Message)
			fmt.Printf("   Offending:   %s\n", v.OffendingNode)
			fmt.Printf("   Remediation: %s\n", getRemediationDescription(v))
			if v.MathematicalProof != "" {
				fmt.Printf("   Proof:       %s\n", v.MathematicalProof)
			}
			fmt.Println()
		}
	}

	if len(rep.Mutations) > 0 {
		fmt.Printf("AST Mutations Applied (%d):\n", len(rep.Mutations))
		for i, m := range rep.Mutations {
			fmt.Printf("%d. [%s] %s\n", i+1, m.ProbeName, m.Description)
		}
		fmt.Println()
	}

	if rep.RewrittenSQL != "" {
		fmt.Println("Certified Rewritten SQL:")
		fmt.Println(rep.RewrittenSQL)
		fmt.Println()
	}

	if rep.Voucher != nil {
		fmt.Printf("Voucher ID:  %s\n", rep.Voucher.VoucherID)
		fmt.Printf("Signature:   %s (%s)\n", rep.Voucher.Signature.Signature[:16]+"...", rep.Voucher.Signature.Algorithm)
	}
	fmt.Println("-----------------------------------------------------------")
}

func getRemediationDescription(v probes.Violation) string {
	if v.Remediation != "" {
		return v.Remediation
	}
	return "Manual query review required"
}
