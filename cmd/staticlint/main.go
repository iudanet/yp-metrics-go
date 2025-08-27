// Пакет main реализует инструмент multichecker для статического анализа кода.
//
// Этот инструмент объединяет несколько статических анализаторов в одну команду
// для проверки Go-кода на потенциальные проблемы, ошибки и стилевые нарушения.
//
// Включенные анализаторы:
//   - Стандартные анализаторы из golang.org/x/tools/go/analysis/passes
//   - Все анализаторы класса SA из пакета staticcheck.io
//   - Дополнительные выбранные анализаторы из staticcheck.io (не SA класса)
//   - Публичные анализаторы из сторонних пакетов
//   - Пользовательский анализатор, запрещающий прямые вызовы os.Exit в функции main пакета main
//
// Использование:
//
//	go run cmd/staticlint/main.go ./...
//
// Также можно собрать и установить инструмент:
//
//	go install github.com/iudanet/yp-metrics-go/cmd/staticlint
//	staticlint ./...
//
// Конфигурация считывается из файла "config.json", расположенного в текущей
// или родительских директориях. Формат конфигурационного файла:
//
//	{
//	  "staticcheck": ["ST1000", "U1000", "QF1001"],
//	  "disabled": ["fieldalignment", "shadow"]
//	}
//
// Массив "staticcheck" включает дополнительные анализаторы staticcheck помимо SA класса.
// Массив "disabled" отключает указанные анализаторы по имени.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/iudanet/yp-metrics-go/cmd/staticlint/exitcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

// configFileName defines the configuration filename.
const configFileName = "config.json"

// config holds the config file structure for enabling/disabling analyzers.
type config struct {
	Staticcheck []string `json:"staticcheck"` // Additional non-SA staticcheck analyzers to enable
	Disabled    []string `json:"disabled"`    // Analyzer names to disable
}

func main() {
	// Prepare standard analyzers from golang.org/x/tools/go/analysis/passes
	standard := []*analysis.Analyzer{
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildssa.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		errorsas.Analyzer,
		fieldalignment.Analyzer,
		framepointer.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		inspect.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shadow.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		sortslice.Analyzer,
		stdmethods.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		testinggoroutine.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		usesgenerics.Analyzer,
	}

	// Custom analyzer(s)
	custom := []*analysis.Analyzer{
		exitcheck.Analyzer,
	}

	// Public third-party analyzers - choose 2+
	public := []*analysis.Analyzer{
		bodyclose.Analyzer,               // HTTP response body close checker
		stylecheck.Analyzers[0].Analyzer, // Style checks from stylecheck package (ST class)
	}

	// Load configuration
	cfg := loadConfig()

	// Check if config is empty (no staticcheck and no disabled entries)
	isConfigEmpty := len(cfg.Staticcheck) == 0 && len(cfg.Disabled) == 0

	disabled := make(map[string]bool)
	for _, name := range cfg.Disabled {
		disabled[name] = true
	}
	enabledStaticcheck := make(map[string]bool)
	for _, id := range cfg.Staticcheck {
		enabledStaticcheck[id] = true
	}

	// Prepare final analyzer list
	var analyzers []*analysis.Analyzer

	if isConfigEmpty {
		// Config is empty or missing: enable all standard + all SA + all public + all custom
		analyzers = append(analyzers, standard...)
		analyzers = append(analyzers, custom...)
		analyzers = append(analyzers, public...)
		for _, s := range staticcheck.Analyzers {
			analyzers = append(analyzers, s.Analyzer)
		}
	} else {
		// Config present: Run only enabled non-disabled analyzers
		for _, a := range standard {
			if !disabled[a.Name] {
				analyzers = append(analyzers, a)
			}
		}
		for _, a := range custom {
			if !disabled[a.Name] {
				analyzers = append(analyzers, a)
			}
		}
		for _, a := range public {
			if !disabled[a.Name] {
				analyzers = append(analyzers, a)
			}
		}

		// Add SA class staticcheck analyzers if not disabled
		for _, s := range staticcheck.Analyzers {
			name := s.Analyzer.Name
			if strings.HasPrefix(name, "SA") && !disabled[name] {
				analyzers = append(analyzers, s.Analyzer)
			}
		}

		// Add non-SA enabled from config and not disabled
		for _, s := range staticcheck.Analyzers {
			name := s.Analyzer.Name
			if !strings.HasPrefix(name, "SA") && enabledStaticcheck[name] && !disabled[name] {
				analyzers = append(analyzers, s.Analyzer)
			}
		}
	}
	if len(analyzers) == 0 {
		panic("no analyzers enabled, please review config.json")
	}

	// Run the multichecker with selected analyzers
	multichecker.Main(analyzers...)
}

// loadConfig loads the config json from current or parent directories or returns empty config.
func loadConfig() config {
	path := findConfigFile()
	data, err := os.ReadFile(path)
	if err != nil {
		// Return empty config if config file not found or unreadable
		return config{}
	}
	var cfg config
	if err := json.Unmarshal(data, &cfg); err != nil {
		// On error parsing, return empty config
		return config{}
	}
	return cfg
}

// findConfigFile looks for config.json up to 3 parent directories.
func findConfigFile() string {
	// Check current directory first
	if _, err := os.Stat(configFileName); err == nil {
		return configFileName
	}
	// Check up to 3 parents
	dir, err := os.Getwd()
	if err != nil {
		return configFileName // fallback
	}
	for i := 0; i < 3; i++ {
		dir = filepath.Dir(dir)
		p := filepath.Join(dir, configFileName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return configFileName
}
