// Command staticlint — мультианализатор проекта rtmetrics.
//
// # Запуск
//
// Из корня модуля:
//
//	go run ./cmd/staticlint ./...
//	go build -o staticlint ./cmd/staticlint && ./staticlint ./...
//
// Аргументы те же, что у unitchecker/multichecker: пакеты в нотации Go
// (`./...`, `.`, путь к пакету). Флаг `-help` печатает список включённых
// анализаторов. Конкретную проверку можно отключить:
//
//	go run ./cmd/staticlint -noosexit=false ./...
//
// # Состав
//
//  1. Стандартные анализаторы golang.org/x/tools/go/analysis/passes.
//  2. Все анализаторы класса SA пакета honnef.co/go/tools/staticcheck.
//  3. Анализаторы других классов staticcheck: S1009 (simple) и ST1012 (stylecheck).
//  4. Публичные анализаторы bodyclose и nilerr.
//  5. Собственный анализатор noosexit.
//
// # golang.org/x/tools/go/analysis/passes
//
//   - appends — некорректный append
//   - asmdecl — несоответствие Go и ассемблерных объявлений
//   - assign — бесполезные присваивания
//   - atomic — неверное использование sync/atomic
//   - atomicalign — неверное выравнивание полей для atomic
//   - bools — ошибки в булевых выражениях
//   - buildtag — неверные build tag
//   - cgocall — нарушения правил cgo
//   - composite — незаданные ключи в составных литералах unkeyed
//   - copylock — копирование значений с блокировками
//   - deepequalerrors — сравнение ошибок через reflect.DeepEqual
//   - defers — подозрительные defer (например Lock без Unlock)
//   - directive — неверные директивы //go:
//   - errorsas — неверный второй аргумент errors.As
//   - hostport — net.JoinHostPort вместо fmt.Sprintf("%s:%d")
//   - httpmux — неверные шаблоны ServeMux (Go 1.22+)
//   - httpresponse — проверка ошибки до использования http.Response
//   - ifaceassert — невозможные приведения интерфейсов
//   - loopclosure — захват переменной цикла в горутине/отложенном вызове
//   - lostcancel — потерянный context.CancelFunc
//   - nilfunc — сравнение функции с nil
//   - nilness — разыменование nil и невозможные сравнения с nil
//   - printf — несоответствие форматной строки и аргументов
//   - reflectvaluecompare — сравнение reflect.Value через == / DeepEqual
//   - shift — сдвиги, выходящие за разрядность
//   - sigchanyzer — небуферизованный канал для signal.Notify
//   - slog — неверные пары ключ/значение slog
//   - sortslice — sort.Slice с неверным типом
//   - sqlrowserr — не проверен Rows.Err после итерации
//   - stdmethods — неверные сигнатуры стандартных методов (WriteTo, Error и т.д.)
//   - stdversion — символы стандартной библиотеки новее указанного go в go.mod
//   - stringintconv — подозрительное преобразование int в string
//   - structtag — неверные struct tag
//   - testinggoroutine — вызов t.Fatal из посторонней горутины
//   - tests — ошибки в тестах и примерах
//   - timeformat — неверный эталон времени в time.Format
//   - unmarshal — Unmarshal в не-указатель
//   - unreachable — недостижимый код
//   - unsafeptr — неверное преобразование uintptr в unsafe.Pointer
//   - unusedresult — игнорирование результата функций вроде fmt.Sprintf, errors.New
//   - unusedwrite — запись в поле, которое никто не читает
//   - waitgroup — Add вызывается внутри горутины, которую Wait уже ждёт
//
// Не включены вспомогательные fact-анализаторы (inspect, ctrlflow, buildssa)
// и шумные проверки shadow / fieldalignment.
//
// # staticcheck, класс SA
//
// Подключаются все анализаторы с префиксом SA (корректность):
//
//   - SA1xxx — сомнительные конструкции и потенциальные баги
//   - SA2xxx — конкурентность
//   - SA3xxx — тесты
//   - SA4xxx — бесполезный код
//   - SA5xxx — корректность (nil, ошибки, строки)
//   - SA6xxx — производительность
//   - SA9xxx — сомнительные и устаревшие API
//
// Полный список: https://staticcheck.io/docs/checks/#SA
//
// # staticcheck, другие классы
//
//   - S1009 — лишняя проверка len/nil перед обращением к срезу/карте (simple)
//   - ST1012 — пакетная переменная ошибки должна иметь вид ErrXxx (stylecheck)
//
// # Публичные анализаторы
//
//   - bodyclose (github.com/timakin/bodyclose) — незакрытое тело http.Response
//   - nilerr (github.com/gostaticanalysis/nilerr) — возврат nil-ошибки при ненулевом err
//
// # noosexit
//
// Запрещает прямой вызов os.Exit в функции main пакета main. Код завершения
// нужно возвращать из main через return (или вынести логику в run() и обработать
// ошибку без os.Exit). Вызовы os.Exit в других функциях не проверяются.
// Сгенерированные файлы (в том числе _testmain.go) пропускаются.
package main

import (
	"strings"

	"github.com/d2cTool/rtmetrics/cmd/staticlint/noosexit"
	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpmux"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/nilness"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/sqlrowserr"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
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
	"golang.org/x/tools/go/analysis/passes/waitgroup"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	multichecker.Main(analyzers()...)
}

func analyzers() []*analysis.Analyzer {
	checks := []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		atomicalign.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		deepequalerrors.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		hostport.Analyzer,
		httpmux.Analyzer,
		httpresponse.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		nilness.Analyzer,
		printf.Analyzer,
		reflectvaluecompare.Analyzer,
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		sortslice.Analyzer,
		sqlrowserr.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
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
		waitgroup.Analyzer,
		bodyclose.Analyzer,
		nilerr.Analyzer,
		noosexit.Analyzer,
	}

	for _, a := range staticcheck.Analyzers {
		if strings.HasPrefix(a.Analyzer.Name, "SA") {
			checks = append(checks, a.Analyzer)
		}
	}
	for _, a := range simple.Analyzers {
		if a.Analyzer.Name == "S1009" {
			checks = append(checks, a.Analyzer)
		}
	}
	for _, a := range stylecheck.Analyzers {
		if a.Analyzer.Name == "ST1012" {
			checks = append(checks, a.Analyzer)
		}
	}

	return checks
}
