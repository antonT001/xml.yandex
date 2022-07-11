package main

import (
	"database/sql"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"xml.yandex/internal/models"

	"gopkg.in/yaml.v3"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	var data_keys []models.Keywords

	max_test := 3
	active_account_id := 0
	active_accout := http.DefaultClient
	var config_db models.Database

	data, err := os.ReadFile(".config.cfg")
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	err = yaml.Unmarshal([]byte(data), &config_db)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	dataSourceName := fmt.Sprintf("%v:%v@tcp(%v)/%v", config_db.User_name, config_db.Password, config_db.Host, config_db.Db_name)

	db, err := sql.Open("mysql", dataSourceName)
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
	fmt.Println(time.Now().Format(time.RFC822), "activated_account:", data_accounts[active_account_id].Account_name)

	for {
		time_request := time.Now().Unix()

		data_task, err := request_last_task(db)
		fmt.Println(time.Now().Format(time.RFC822), "request_last_task")
		if err != nil {
			fmt.Println(time.Now().Format(time.RFC822), err)
			creation_new_task(db, time_request) //это вообще легально?
			continue
		}

		if data_task.Date == 0 {
			fmt.Println(time.Now().Format(time.RFC822), "no_last_task")

			creation_new_task(db, time_request)
			fmt.Println(time.Now().Format(time.RFC822), "creation_new_task")

		}

		today := time.Unix(time_request, 0).Day()
		day_on_task := time.Unix(data_task.Date, 0).Day()

		if data_task.Completed == 1 { //завершено, смотрим на его дату,если:

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

			time_request = data_task.Date
			fmt.Println(time.Now().Format(time.RFC822), "copy_time_request_from_task")

			if data_task.Primary_check == 0 {
				fmt.Println(time.Now().Format(time.RFC822), "primary_check_not_completed")

				data_keys, err = request_keywords(db, data_task.Last_processed_key_id)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Printf(time.Now().Format(time.RFC822)+" request_keywords(%v)\n", data_task.Last_processed_key_id)

			} else {
				fmt.Println(time.Now().Format(time.RFC822), "primary_check_completed")

				data_keys, err = request_statistics_zero_positions(db, time_request, max_test)
				if err != nil {
					log.Fatal("request_keywords(): ", err)
				}
				fmt.Println(time.Now().Format(time.RFC822), "request_statistics_zero_positions")

				if len(data_keys) < 1 {
					fmt.Println(time.Now().Format(time.RFC822), "all_zero_positions_processed")

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
				data_accounts[active_account_id].Account_name, data_accounts[active_account_id].Account_key,
				url.QueryEscape(data.Keyword_name)) /////

			respData := GetRequest(url+"groups-on-page%3D100", active_accout) ////

			v := models.Result{}

			xml.Unmarshal(respData, &v)
			fmt.Println(time.Now().Format(time.RFC822), "GetRequest ", data.Keyword_name)

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

			var vol models.Group

			result_found := false
			for i, vol := range result {
				if vol.Host_name == data.Host_name {
					if data.In_statistics == 0 { //новое значение в базу
						fmt.Printf(time.Now().Format(time.RFC822)+" keyword_found_at_position_%v,_write_to_the_database\n", i)

						insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
						VALUES (?, ?, ?, ?, ?);`, i+1, vol.Url, time_request, data.Host_id, data.Keyword_id)
						if err != nil {
							log.Fatal("INSERT INTO statistics ", err)
						}
						insert.Close()

						insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.Keyword_id, time_request)
						if err != nil {
							log.Fatal("UPDATE task ", err)
						}
						insert.Close()

					} else { //старое значения в базе
						fmt.Printf(time.Now().Format(time.RFC822)+" keyword_found_at_position_%v,_update_the_value_in_the_database\n", i)

						insert, err := db.Query(`UPDATE statistics SET position_num = ?, url = ?, test_number=test_number+1 
						WHERE keyword_id=? && date = ?;`, i+1, vol.Url, data.Keyword_id, time_request)
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
				if data.In_statistics == 0 {
					fmt.Println(time.Now().Format(time.RFC822), "keyword_not_found_write_position_zero_to_database")

					insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
					VALUES (?, ?, ?, ?, ?)`, 0, vol.Url, time_request, data.Host_id, data.Keyword_id)
					if err != nil {
						log.Fatal("INSERT INTO statistics when position_num=0", err)
					}
					insert.Close()
					insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.Keyword_id, time_request)
					if err != nil {
						log.Fatal("UPDATE task ", err)
					}
					insert.Close()
				} else {
					fmt.Println(time.Now().Format(time.RFC822), "keyword_not_found_updating_test_information")

					insert, err := db.Query(`UPDATE statistics SET test_number=test_number+1 WHERE keyword_id=? && date = ?;`, data.Keyword_id, time_request)
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

func request_last_task(db *sql.DB) (models.Task, error) {
	row := db.QueryRow(`SELECT date, last_processed_key_id, primary_check, completed 
	FROM task ORDER BY id DESC LIMIT 1;`)

	var data_task models.Task

	if err := row.Scan(&data_task.Date, &data_task.Last_processed_key_id, &data_task.Primary_check,
		&data_task.Completed); err != nil {
		return data_task, fmt.Errorf("request_last_task() %v", err) //завязано на логику
	}

	if err := row.Err(); err != nil {
		log.Fatal("request_last_task row.Err() ", err)
	}

	return data_task, nil
}

func request_keywords(db *sql.DB, n int) ([]models.Keywords, error) {
	var data_keys []models.Keywords
	rows, err := db.Query(`SELECT keywords.id, keywords.keyword_name, hosts.id,hosts.host_name FROM keywords 
	LEFT JOIN hosts ON hosts.id=keywords.host_id WHERE keywords.id > ?;`, n)
	if err != nil {
		log.Fatal("request_keywords()", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Keywords
		if err := rows.Scan(&data.Keyword_id, &data.Keyword_name, &data.Host_id, &data.Host_name); err != nil {
			return nil, fmt.Errorf("request_keywords %v", err)
		}
		data_keys = append(data_keys, data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("request_keywords %v", err)
	}
	return data_keys, nil
}

func request_statistics_zero_positions(db *sql.DB, time_request int64, max_test int) ([]models.Keywords, error) {
	var data_keys []models.Keywords

	rows, err := db.Query(`SELECT statistics.keyword_id, keywords.keyword_name,hosts.id,hosts.host_name FROM statistics 
	LEFT JOIN hosts ON hosts.id=statistics.host_id 
	LEFT JOIN keywords ON keywords.id=statistics.keyword_id 
	WHERE position_num = 0 && date = ? && test_number< ?;`, time_request, max_test)
	if err != nil {
		log.Fatal("equest_statistics_zero_positions()", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Keywords
		if err := rows.Scan(&data.Keyword_id, &data.Keyword_name, &data.Host_id, &data.Host_name); err != nil {
			return nil, fmt.Errorf("equest_statistics_zero_positions() %v", err)
		}
		data.In_statistics = 1
		data_keys = append(data_keys, data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("equest_statistics_zero_positions() %v", err)
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

func account_change(active_accout_id int, data_accounts []models.Account) (int, *http.Client) {
	if active_accout_id+1 < len(data_accounts) {
		active_accout_id++

		proxyUrl, _ := url.Parse(fmt.Sprintf(`http://%v:%v@%v`, data_accounts[active_accout_id].Proxy_login,
			data_accounts[active_accout_id].Proxy_password, data_accounts[active_accout_id].Proxy_ip))

		myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}
		fmt.Println(time.Now().Format(time.RFC822), "activated_account:", data_accounts[active_accout_id].Account_name)

		return active_accout_id, myClient
	}
	fmt.Println(time.Now().Format(time.RFC822), "activated_account:", data_accounts[0].Account_name)

	return 0, http.DefaultClient
}

func request_accounts(db *sql.DB) ([]models.Account, error) {
	var data_accounts []models.Account
	rows, err := db.Query(`SELECT * FROM accounts;`)
	if err != nil {
		log.Fatal("request_accounts()", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Account
		if err := rows.Scan(&data.Id, &data.Account_name, &data.Account_key, &data.Proxy_ip,
			&data.Proxy_login, &data.Proxy_password); err != nil {
			return nil, fmt.Errorf("request_accounts() %v", err)
		}
		data_accounts = append(data_accounts, data)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("request_accounts() %v", err)
	}
	return data_accounts, nil
}
