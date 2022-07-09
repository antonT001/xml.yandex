package main

//https://yandex.ru/search/xml?action=limits-info&user=seo-art-spectrum&key=03.437953978:a79c783413e3d0d205a60ce7ea6762c5
import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	var data_keys []Keywords

	max_test := 5

	// f, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_WRONLY, 0600)
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()

	db, err := sql.Open("mysql", "user:123456@tcp(127.0.0.1:8080)/xml")
	if err != nil {
		log.Fatal("sql.Open ", err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println(time.Now().Format(time.RFC822), "Connected_db!")
	defer db.Close()

	for {
		time_request := time.Now().Unix()

		data_task, err := request_last_task(db)
		fmt.Println(time.Now().Format(time.RFC822), "request_last_task")
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC822), err)
			creation_new_task(db, time_request) //это вообще легально?
			continue
		}

		if data_task.date == 0 {
			fmt.Println(time.Now().Format(time.RFC822), "no_last_task")

			creation_new_task(db, time_request)
			fmt.Println(time.Now().Format(time.RFC822), "creation_new_task")

		}

		today := time.Unix(time_request, 0).Day()
		day_on_task := time.Unix(data_task.date, 0).Day()

		if data_task.completed == 1 { //завершено, смотрим на его дату,если:

			if today != day_on_task { //не сегодня, то создаем новое задание,
				fmt.Println(time.Now().Format(time.RFC822), "last_task_completed_not_today")

				creation_new_task(db, time_request)
				fmt.Println(time.Now().Format(time.RFC822), "creation_new_task")

				data_keys, err = request_keywords(db, 0)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Println(time.Now().Format(time.RFC822), "request_keywords(0)")

			} else { //сегодня, сделать паузу до конца суток
				fmt.Println(time.Now().Format(time.RFC822), "last_task_completed_today")

				time.Sleep(time.Minute * 30)                                    //////
				fmt.Println(time.Now().Format(time.RFC822), "Sleep 30 minutes") //////

				continue
			}

		} else {
			fmt.Println(time.Now().Format(time.RFC822), "last_task_not_completed")

			if data_task.primary_check == 0 {
				fmt.Println(time.Now().Format(time.RFC822), "primary_check_not_completed")

				time_request = data_task.date
				fmt.Println(time.Now().Format(time.RFC822), "copy_time_request_from_task")

				data_keys, err = request_keywords(db, data_task.last_processed_key_id)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Println(time.Now().Format(time.RFC822), "request_keywords(", data_task.last_processed_key_id, ")")

			} else {
				fmt.Println(time.Now().Format(time.RFC822), "primary_check_completed")

				data_keys, err = request_statistics_0_positions(db, max_test)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Println(time.Now().Format(time.RFC822), "request_statistics_0_positions")

				if len(data_keys) < 1 {
					fmt.Println(time.Now().Format(time.RFC822), "all_0_positions_processed")

					insert, err := db.Query(`UPDATE task SET completed = 1, WHERE date = ?;`, time_request)
					if err != nil {
						log.Fatal("UPDATE task completed", err)
					}
					defer insert.Close()
					fmt.Println(time.Now().Format(time.RFC822), "task_completion")

					continue
				}
			}
		}

		fmt.Println(time.Now().Format(time.RFC822), "key_processing...")

		error_flag := false

		for _, data := range data_keys {

			var url = `https://yandex.ru/search/xml?user=seo-art-spectrum&key=03.437953978:a79c783413e3d0d205a60ce7ea6762c5&lr=2&query=` + url.QueryEscape(data.keyword_name) + `&groupby=groups-on-page%3D100`

			respData := GetRequest(url)

			v := Result{}

			xml.Unmarshal(respData, &v)

			response := v.Response

			if response.Error.Error_code != 0 {
				fmt.Println(time.Now().Format(time.RFC822), "error code = ", response.Error.Error_code)
				//https://yandex.ru/dev/xml/doc/dg/reference/error-codes.html#error-codes
				fmt.Println(time.Now().Format(time.RFC822), "sleep 5 minutes")
				time.Sleep(time.Minute * 5)
				error_flag = true
				break
			}

			result := v.Response.Results.Grouping.Group

			var vol Group

			result_found := false
			for i, vol := range result {
				if vol.Host_name == data.host_name {
					if data.in_statistics == 0 { //новое значение в базу
						//плохо, что у меня сделано два запроса. Можно объеденить в один? В данной логике важно, чтоб они выполнились
						//оба или ниодного
						insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
						VALUES (?, ?, ?, ?, ?);`, i+1, vol.Url, time_request, data.host_id, data.keyword_id)
						if err != nil {
							log.Fatal("INSERT INTO statistics ", err)
						}
						defer insert.Close()    //нужно два?
						time.Sleep(time.Second) //Error 1040: Too many connections
						insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.keyword_id, time_request)
						if err != nil {
							log.Fatal("UPDATE task ", err)
						}
						defer insert.Close()    //нужно 2?
						time.Sleep(time.Second) //Error 1040: Too many connections
					} else { //старое значения в базе
						insert, err := db.Query(`UPDATE statistics SET position_num = ?, test_number=test_number+1 
						WHERE keyword_id=? && date = ?;`, i+1, data.keyword_id, time_request)
						if err != nil {
							log.Fatal("UPDATE statistics", err)
						}
						defer insert.Close()
					}
					result_found = true
					break
				}
			}

			if !result_found {
				if data.in_statistics == 0 {
					insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
					VALUES (?, ?, ?, ?, ?)`, 0, vol.Url, time_request, data.host_id, data.keyword_id)
					if err != nil {
						log.Fatal("INSERT INTO statistics when position_num=0", err)
					}
					defer insert.Close()
				}
			}
			time.Sleep(time.Second)
		}

		if !error_flag {
			//отметить в базе, что первичный проход выполнен
			insert, err := db.Query(`UPDATE task SET primary_check = 1 WHERE date = ?;`, time_request)
			if err != nil {
				log.Fatal("UPDATE statistics", err)
			}
			defer insert.Close()
			fmt.Println(time.Now().Format(time.RFC822), "primary_check_completed")
		}

	}

}

type Task struct {
	date                  int64
	last_processed_key_id int
	primary_check         int
	completed             int
}

type Keywords struct {
	keyword_id    int
	keyword_name  string
	host_id       int
	host_name     string
	in_statistics int
}

type Group struct {
	Host_name string `xml:"doc>domain"`
	Url       string `xml:"doc>url"`
}

type Grouping struct {
	Page  int     `xml:"page"`
	Group []Group `xml:"group"`
}

type Results struct {
	Grouping Grouping `xml:"grouping"`
}
type Error struct {
	Error_code int `xml:"code,attr"`
}

type Response struct {
	Error   Error   `xml:"error"`
	Results Results `xml:"results"`
}

type Result struct {
	Response Response `xml:"response"`
}

func GetRequest(url string) []byte {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return respData
}

func request_last_task(db *sql.DB) (Task, error) {
	row := db.QueryRow(`SELECT date, last_processed_key_id, primary_check, completed 
	FROM task ORDER BY id DESC LIMIT 1;`)

	var data_task Task

	if err := row.Scan(&data_task.date, &data_task.last_processed_key_id, &data_task.primary_check,
		&data_task.completed); err != nil {
		return data_task, fmt.Errorf("request_last_task() %v", err) //завязано на логику
	}

	if err := row.Err(); err != nil {
		log.Fatal("request_last_task row.Err() ", err)
	}

	return data_task, nil
}

func request_keywords(db *sql.DB, n int) ([]Keywords, error) {
	var data_keys []Keywords
	rows, err := db.Query(`SELECT keywords.id, keywords.keyword_name, hosts.id,hosts.host_name FROM keywords 
	LEFT JOIN hosts ON hosts.id=keywords.host_id WHERE keywords.id > ?;`, n)
	if err != nil {
		log.Fatal("request_keywords()", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data Keywords
		if err := rows.Scan(&data.keyword_id, &data.keyword_name, &data.host_id, &data.host_name); err != nil {
			return nil, fmt.Errorf("request_keywords %v", err)
		}
		data_keys = append(data_keys, data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("request_keywords %v", err)
	}
	return data_keys, nil
}

func request_statistics_0_positions(db *sql.DB, max_test int) ([]Keywords, error) {
	var data_keys []Keywords

	rows, err := db.Query(`SELECT statistics.keyword_id, keywords.keyword_name,hosts.id,hosts.host_name FROM statistics 
	LEFT JOIN hosts ON hosts.id=statistics.host_id 
	LEFT JOIN keywords ON keywords.id=statistics.keyword_id 
	WHERE position_num = 0 && test_number<= ?;`, max_test)
	if err != nil {
		log.Fatal("equest_statistics_0_positions()", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data Keywords
		if err := rows.Scan(&data.keyword_id, &data.keyword_name, &data.host_id, &data.host_name); err != nil {
			return nil, fmt.Errorf("equest_statistics_0_positions() %v", err)
		}
		data.in_statistics = 1
		data_keys = append(data_keys, data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("equest_statistics_0_positions() %v", err)
	}
	return data_keys, nil
}

func creation_new_task(db *sql.DB, time_request int64) {
	insert, err := db.Query(`INSERT INTO task (date) VALUES (?);`, time_request)
	if err != nil {
		log.Fatal("INSERT INTO task date ", err)
	}
	defer insert.Close()
}
