package main

//https://yandex.ru/search/xml?action=limits-info&user=seo-art-spectrum&key=03.437953978:a79c783413e3d0d205a60ce7ea6762c5

//http://user:password2@144.76.91.205:1010/yandex.ru/search/xml?user=migunowvad&key=03.1046407880:1a97fb53f91282cbd5dddc7fb48b8f25&lr=2&query=планкен%20купитьgroupby=groups-on-page%3D100

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

	max_test := 3
	active_account_id := 0
	active_accout := http.DefaultClient

	db, err := sql.Open("mysql", "user:123456@tcp(127.0.0.1:8080)/xml")
	if err != nil {
		log.Fatal("sql.Open ", err)
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxIdleTime(time.Second * 10)
	db.SetConnMaxLifetime(time.Second * 10)

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println(time.Now().Format(time.RFC822), "connected_db")
	defer db.Close()

	data_accounts, err := request_accounts(db)
	if err != nil {
		log.Fatal("request_accounts", err)
	}
	fmt.Println(time.Now().Format(time.RFC822), "request_accounts")
	fmt.Println(time.Now().Format(time.RFC822), "activated_account:", data_accounts[active_account_id].account_name)

	/*
		data_accounts, err := request_accounts(db)
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC822), err)
		}
		fmt.Println(time.Now().Format(time.RFC822), "request_accounts")


		///
		fmt.Println("data_accounts= ", data_accounts)
		///
		///
		fmt.Println("active_accout= ", active_accout)
		///
		active_accout, err = account_change(active_accout, data_accounts)
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC822), err)
		}
		fmt.Println(time.Now().Format(time.RFC822), "account_change")
		///
		fmt.Println("active_accout= ", active_accout)
		///
		active_accout, err = account_change(active_accout, data_accounts)
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC822), err)
		}
		fmt.Println(time.Now().Format(time.RFC822), "account_change")
		///
		fmt.Println("active_accout= ", active_accout)
		///
		active_accout, err = account_change(active_accout, data_accounts)
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC822), err)
		}
		fmt.Println(time.Now().Format(time.RFC822), "account_change")
		///
		fmt.Println("active_accout= ", active_accout)
		///

		///
		time.Sleep(time.Minute)
		///
	*/

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
				
				fmt.Println(time.Now().Format(time.RFC822), "Sleep 30 minutes") //////
				time.Sleep(time.Minute * 30)                                    //////
				
				continue
			}

		} else {
			fmt.Println(time.Now().Format(time.RFC822), "last_task_not_completed")

			time_request = data_task.date
			fmt.Println(time.Now().Format(time.RFC822), "copy_time_request_from_task")

			if data_task.primary_check == 0 {
				fmt.Println(time.Now().Format(time.RFC822), "primary_check_not_completed")

				data_keys, err = request_keywords(db, data_task.last_processed_key_id)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Printf(time.Now().Format(time.RFC822)+ " request_keywords(%v)\n", data_task.last_processed_key_id)

			} else {
				fmt.Println(time.Now().Format(time.RFC822), "primary_check_completed")

				data_keys, err = request_statistics_0_positions(db, time_request, max_test)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Println(time.Now().Format(time.RFC822), "request_statistics_0_positions")

				if len(data_keys) < 1 {
					fmt.Println(time.Now().Format(time.RFC822), "all_0_positions_processed")

					insert, err := db.Query(`UPDATE task SET completed = 1 WHERE date = ?;`, time_request)
					if err != nil {
						log.Fatal("UPDATE task completed", err)
					}
					insert.Close()
					fmt.Println(time.Now().Format(time.RFC822), "task_completion")

					continue
				}
			}
		}

		fmt.Println(time.Now().Format(time.RFC822), "keywords_processing...")

		error_flag := false

		for _, data := range data_keys {

			var url = fmt.Sprintf("https://yandex.ru/search/xml?user=%v&key=%v&lr=2&query=%v&groupby=",
				data_accounts[active_account_id].account_name, data_accounts[active_account_id].account_key,
				url.QueryEscape(data.keyword_name)) /////

			respData := GetRequest(url+"groups-on-page%3D100", active_accout) ////

			v := Result{}

			xml.Unmarshal(respData, &v)
			fmt.Println(time.Now().Format(time.RFC822),"GetRequest ", data.keyword_name)

			response := v.Response

			if response.Error.Error_code != 0 {
				fmt.Println(time.Now().Format(time.RFC822), "error code = ", response.Error.Error_code)
				//https://yandex.ru/dev/xml/doc/dg/reference/error-codes.html#error-codes

				active_account_id, active_accout = account_change(active_account_id, data_accounts)
				fmt.Println(time.Now().Format(time.RFC822), "account_change")

				if active_account_id == 0 {
					data_accounts, err = request_accounts(db)
					if err != nil {
						log.Fatal("request_accounts", err)
					}
					fmt.Println(time.Now().Format(time.RFC822), "sleep_5_minutes")
					time.Sleep(time.Minute * 5)
					fmt.Println(time.Now().Format(time.RFC822), "request_accounts")
				}

				error_flag = true /// проверить необходимость
				break
			}

			result := v.Response.Results.Grouping.Group

			var vol Group

			result_found := false
			for i, vol := range result {
				if vol.Host_name == data.host_name {
					if data.in_statistics == 0 { //новое значение в базу
						fmt.Printf(time.Now().Format(time.RFC822)+" keyword_found_at_position_%v,_write_to_the_database\n", i)

						insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
						VALUES (?, ?, ?, ?, ?);`, i+1, vol.Url, time_request, data.host_id, data.keyword_id)
						if err != nil {
							log.Fatal("INSERT INTO statistics ", err)
						}
						insert.Close()

						insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.keyword_id, time_request)
						if err != nil {
							log.Fatal("UPDATE task ", err)
						}
						insert.Close()

					} else { //старое значения в базе
						fmt.Printf(time.Now().Format(time.RFC822)+" keyword_found_at_position_%v,_update_the_value_in_the_database\n", i)

						insert, err := db.Query(`UPDATE statistics SET position_num = ?, url = ?, test_number=test_number+1 
						WHERE keyword_id=? && date = ?;`, i+1, vol.Url, data.keyword_id, time_request)
						if err != nil {
							log.Fatal("UPDATE statistics", err)
						}
						insert.Close()
					}
					result_found = true
					break
				}
			}

			if !result_found {
				if data.in_statistics == 0 {
					fmt.Println(time.Now().Format(time.RFC822), "keyword_not_found_write_position_0_to_database")

					insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
					VALUES (?, ?, ?, ?, ?)`, 0, vol.Url, time_request, data.host_id, data.keyword_id)
					if err != nil {
						log.Fatal("INSERT INTO statistics when position_num=0", err)
					}
					insert.Close()
					insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.keyword_id, time_request)
					if err != nil {
						log.Fatal("UPDATE task ", err)
					}
					insert.Close()
				} else {
					fmt.Println(time.Now().Format(time.RFC822), "keyword_not_found_updating_test_information")

					insert, err := db.Query(`UPDATE statistics SET test_number=test_number+1 WHERE keyword_id=? && date = ?;`, data.keyword_id, time_request)
					if err != nil {
						log.Fatal("UPDATE statistics", err)
					}
					insert.Close()
				}
			}
			time.Sleep(time.Second)
		}

		if !error_flag {
			//первичный проход выполнен
			insert, err := db.Query(`UPDATE task SET primary_check = 1 WHERE date = ?;`, time_request)
			if err != nil {
				log.Fatal("UPDATE statistics", err)
			}
			insert.Close()
			fmt.Println(time.Now().Format(time.RFC822), "primary_check_completed")
		}

	}

}

type Account struct {
	id             int
	account_name   string
	account_key    string
	proxy_ip       string
	proxy_login    string
	proxy_password string
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

func GetRequest(url string, active_accout *http.Client) []byte {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := active_accout.Do(req)
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

func request_statistics_0_positions(db *sql.DB, time_request int64, max_test int) ([]Keywords, error) {
	var data_keys []Keywords

	rows, err := db.Query(`SELECT statistics.keyword_id, keywords.keyword_name,hosts.id,hosts.host_name FROM statistics 
	LEFT JOIN hosts ON hosts.id=statistics.host_id 
	LEFT JOIN keywords ON keywords.id=statistics.keyword_id 
	WHERE position_num = 0 && date = ? && test_number< ?;`, time_request, max_test)
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
	insert.Close()
}

func account_change(active_accout_id int, data_accounts []Account) (int, *http.Client) {
	if active_accout_id+1 < len(data_accounts) {
		active_accout_id++

		proxyUrl, _ := url.Parse(fmt.Sprintf(`http://%v:%v@%v`, data_accounts[active_accout_id].proxy_login,
			data_accounts[active_accout_id].proxy_password, data_accounts[active_accout_id].proxy_ip))

		myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
		fmt.Println(time.Now().Format(time.RFC822), "activated_account:", data_accounts[active_accout_id].account_name)

		return active_accout_id, myClient
	}
	fmt.Println(time.Now().Format(time.RFC822), "activated_account:", data_accounts[0].account_name)

	return 0, http.DefaultClient
}

func request_accounts(db *sql.DB) ([]Account, error) {
	var data_accounts []Account
	rows, err := db.Query(`SELECT * FROM accounts;`)
	if err != nil {
		log.Fatal("request_accounts()", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data Account
		if err := rows.Scan(&data.id, &data.account_name, &data.account_key, &data.proxy_ip,
			&data.proxy_login, &data.proxy_password); err != nil {
			return nil, fmt.Errorf("request_accounts() %v", err)
		}
		data_accounts = append(data_accounts, data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("request_accounts() %v", err)
	}
	return data_accounts, nil
}
