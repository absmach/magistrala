package bone

/*
import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/daryl/zeus"
	"github.com/gorilla/mux"
	"github.com/gorilla/pat"
	"github.com/julienschmidt/httprouter"
	"github.com/ursiform/bear"
)

// Test the ns/op
func BenchmarkBoneMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd", nil)
	response := httptest.NewRecorder()
	muxx := New()

	muxx.Get("/", http.HandlerFunc(Bench))
	muxx.Get("/a", http.HandlerFunc(Bench))
	muxx.Get("/aas", http.HandlerFunc(Bench))
	muxx.Get("/sd", http.HandlerFunc(Bench))

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

// Test httprouter ns/op
func BenchmarkHttpRouterMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd", nil)
	response := httptest.NewRecorder()
	muxx := httprouter.New()

	muxx.Handler("GET", "/", http.HandlerFunc(Bench))
	muxx.Handler("GET", "/a", http.HandlerFunc(Bench))
	muxx.Handler("GET", "/aas", http.HandlerFunc(Bench))
	muxx.Handler("GET", "/sd", http.HandlerFunc(Bench))

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

// Test daryl/zeus ns/op
func BenchmarkZeusMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd/test", nil)
	response := httptest.NewRecorder()
	muxx := zeus.New()

	muxx.GET("/", Bench)
	muxx.GET("/a", Bench)
	muxx.GET("/aas", Bench)
	muxx.GET("/sd/:id", Bench)

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

// Test net/http ns/op
func BenchmarkNetHttpMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd", nil)
	response := httptest.NewRecorder()
	muxx := http.NewServeMux()

	muxx.HandleFunc("/", Bench)
	muxx.HandleFunc("/a", Bench)
	muxx.HandleFunc("/aas", Bench)
	muxx.HandleFunc("/sd", Bench)

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

// Test ursiform/bear ns/op
func BenchmarkBearMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd", nil)
	response := httptest.NewRecorder()
	muxx := bear.New()

	muxx.On("GET", "/", Bench)
	muxx.On("GET", "/a", Bench)
	muxx.On("GET", "/aas", Bench)
	muxx.On("GET", "/sd", Bench)

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

// Test gorilla/mux ns/op
func BenchmarkGorillaMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd", nil)
	response := httptest.NewRecorder()
	muxx := mux.NewRouter()

	muxx.Handle("/", http.HandlerFunc(Bench))
	muxx.Handle("/a", http.HandlerFunc(Bench))
	muxx.Handle("/aas", http.HandlerFunc(Bench))
	muxx.Handle("/sd", http.HandlerFunc(Bench))

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

// Test gorilla/pat ns/op
func BenchmarkGorillaPatMux(b *testing.B) {
	request, _ := http.NewRequest("GET", "/sd", nil)
	response := httptest.NewRecorder()
	muxx := pat.New()

	muxx.Get("/", Bench)
	muxx.Get("/a", Bench)
	muxx.Get("/aas", Bench)
	muxx.Get("/sd", Bench)

	for n := 0; n < b.N; n++ {
		muxx.ServeHTTP(response, request)
	}
}

func Bench(rw http.ResponseWriter, req *http.Request) {
	rw.Write([]byte("b"))
}
*/
/*
			### Result ###

BenchmarkBoneMux				10000000	       124 ns/op
BenchmarkHttpRouterMux			10000000	       147 ns/op
BenchmarkZeusMux				10000000	       210 ns/op
BenchmarkNetHttpMux	 		 	 3000000	       560 ns/op
BenchmarkGorillaMux	  			  500000	      2946 ns/op
BenchmarkGorillaPatMux	 		 1000000	      1805 ns/op

ok  	github.com/go-zoo/bone	10.997s

*/
