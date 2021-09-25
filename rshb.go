package main
// исходный код, ревью которого делается: https://goplay.space/#hMh2XyixXRC
import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
	"github.com/jackc/pgx/v4"
	"time"
)

type BookModel struct {
	Title  string
	Author string
	Cost   int
}

type Service struct {
	Pool   []*pgx.Conn
	IsInit bool
}
// указатель добавляем
func (s *Service) initService(username string, password string) {
	done := make(chan struct{})
	// делаем канал пустышку для синхронизации 
	
	var backgroundTask = func(done chan struct{}) {
		var databaseUrl = "postgres://" + username + ":" + password + "@10.7.27.34:5432/books"

		for i := 1; i <= 10; i++ {
			conn, err := pgx.Connect(context.Background(), databaseUrl)
			if err != nil {
				fmt.Println("Ошибка при подключении к базе по URL = " + databaseUrl)
				panic(nil)
			}
			s.Pool = append(s.Pool, conn)

		}
		close(done)
	}
	go backgroundTask(done)
	<- done

	// синхронизируемся, чтобы не запустить запрос к базе до успешного набора пула подключений
	// потому что проверка на строке 67 нас не спасет, мы пролетим через цикл и
	// ничего не получив начнем выполнение. Можно изменить проверку на соединения,
	// если их нет совсем, то выдать ошибку там. Или круить цикл в бесконечности,
	// пока не получим соединения. реализовал оба способа.
}

func (s *Service) getBooksByAuthor(username, password string, author string, result *[]BookModel) {
	// необходимо вернуть значение result через указатель, также через указатель
	// идем к экземпляру, получать-передавать данные из другой функции

	start := time.Now()

	if !s.IsInit {
		s.initService(username, password)
		s.IsInit = true
	}

	var conn *pgx.Conn
	var check_conn bool

	for check_conn != true {
		for _, x := range s.Pool {
			if !x.IsClosed() {
				conn = x
				check_conn = true // флажок для вечного цикла
				break
			}
		}
		// а еще можно добавить таймаут операции, здесь уже не нужно, но для красоты
		// можно как условие верхнего цикла 
		if time.Since(start).Seconds() > 5 {
			fmt.Println("Не получено ни одного подключения")
			break
		}
	}

	rows, err := conn.Query(context.Background(), "select title, cost from books where author=" + "'" + author + "'") 
	// ошибка в запросе не хватает - "'" вокруг author
	if err != nil {
		fmt.Println("Не удалось получить книги по автору")
		panic(nil)
	}

	for rows.Next() {
		var title string
		var cost int
		err = rows.Scan(&title, &cost)
		if err == nil {
			*result = append(*result, BookModel{title, author, cost})
		}
	}
	fmt.Println("Успешно выполнен запрос, заполнено записей: " + strconv.Itoa(len(*result)))
	// println - некорректно использовать, хотя кто нам мешает при отладке например, но пускать официально неправильно
}

func main() {

	fmt.Println("Запуск сервера...")
	var service = Service{}

	r := mux.NewRouter()
	r.HandleFunc("/GetBookByAuthor/{author}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		author := vars["author"]
		result := make([]BookModel, 0)
		// было инициализировано 10 пустых записей, к которым мы будем добавлять результат в функции, убрал 10

		service.getBooksByAuthor("postgres", "", author, &result)
		//разыменовываем result для передачи в функцию и получаем результат
		w.WriteHeader(http.StatusOK)
		for _, elem := range result { //для развлечения :)
			_, err := fmt.Fprintf(w, "Result: %s | %s | %d \n", elem.Title, elem.Author, elem.Cost)
			if err != nil {
				fmt.Println(err)
			}
		}
	})
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		fmt.Println(err)
	}
	// не было обработчика ошибок
}
