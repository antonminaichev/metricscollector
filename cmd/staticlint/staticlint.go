// Package main реализует multichecker для статического анализа Go-кода.
//
// Multichecker объединяет в себе:
//   - стандартные анализаторы из пакета golang.org/x/tools/go/analysis/passes;
//   - все SA-анализаторы из staticcheck.io (анализаторы ошибок и неправильных конструкций);
//   - по одному анализатору из классов quickfix (QF1003), simple (S1005) и stylecheck (ST1005);
//   - сторонние публичные анализаторы bodyclose и errcheck;
//   - собственный анализатор noexitanalyzer, запрещающий использовать os.Exit в функции main пакета main.
//
// # Как использовать:
//
// Установить multichecker:
//
//	go build -o staticlint ./cmd/staticlint
//
// Запустить анализ:
//
//	./staticlint ./...
//
// Результатом будет список найденных проблем в коде.
//
// # Описание включённых анализаторов:
//
//   - printf, structtag, shadow, unreachable, waitgroup, lostcancel:
//     стандартные анализаторы Go, проверяют распространённые ошибки — забытые cancel'ы, некорректный printf и т.д.
//
//   - SAxxxx (staticcheck):
//     статические проверки на ошибки в коде: лишние проверки, неправильное использование API, утечки, гонки.
//
//   - S1005 (simple):
//     замена выражения `x = x + 1` на `x++`, упрощение синтаксиса.
//
//   - ST1005 (stylecheck):
//     требования к стилю: первая буква комментария к экспортируемому символу должна быть заглавной.
//
//   - QF1003 (quickfix):
//     предлагает замену на strings.Contains вместо strings.Index >= 0.
//
//   - bodyclose:
//     проверяет, что тело http.Response действительно закрывается.
//
//   - errcheck:
//     проверяет, что ошибки не остаются без обработки.
//
//   - noexitanalyzer:
//     собственный анализатор. Запрещает прямой вызов os.Exit в функции main главного пакета.
//     Это сделано для улучшения тестируемости и управления завершением программы.
package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/waitgroup"

	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/antonminaichev/metricscollector/cmd/staticlint/noexitanalyzer"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
)

func main() {
	var analyzers []*analysis.Analyzer

	// Добавляем стандартные анализаторы из passes
	analyzers = append(analyzers,
		printf.Analyzer,
		structtag.Analyzer,
		shadow.Analyzer,
		unreachable.Analyzer,
		waitgroup.Analyzer,
		lostcancel.Analyzer,
	)

	// SA анализаторы staticcheck
	for _, v := range staticcheck.Analyzers {
		if v.Analyzer.Name[:2] == "SA" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// S1005 анализатор
	for _, v := range simple.Analyzers {
		if v.Analyzer.Name == "S1005" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// ST1005 анализатор
	for _, v := range stylecheck.Analyzers {
		if v.Analyzer.Name == "ST1005" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// QF1003 анализатор
	for _, v := range quickfix.Analyzers {
		if v.Analyzer.Name == "QF1003" {
			analyzers = append(analyzers, v.Analyzer)
		}
	}

	// Добавляем сторонние анализаторы
	analyzers = append(analyzers,
		bodyclose.Analyzer,
		errcheck.Analyzer,
	)

	// Собственный анализатор
	analyzers = append(analyzers, noexitanalyzer.Analyzer)

	multichecker.Main(analyzers...)
}
