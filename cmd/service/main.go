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

	_ "github.com/go-sql-driver/mysql"
	"gopkg.in/yaml.v3"
	"xml.yandex/internal/models"
)

func main() {
	var data_keys []models.Keywords

	max_test := 3
	active_account_id := 0
	active_accout := http.DefaultClient
	var config_db models.Database

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stderr, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	for {

		data, err := os.ReadFile(".config.cfg")
		if err != nil {
			errorLog.Println(err)
			infoLog.Println("program_restart")
			time.Sleep(time.Minute)
			continue
		}

		err = yaml.Unmarshal([]byte(data), &config_db)
		if err != nil {
			errorLog.Println(err)
			infoLog.Println("program_restart")
			time.Sleep(time.Minute)
			continue
		}

		dataSourceName := fmt.Sprintf("%v:%v@tcp(%v)/%v", config_db.User_name, config_db.Password, config_db.Host, config_db.Db_name)

		db, err := sql.Open("mysql", dataSourceName)
		if err != nil {
			errorLog.Println(err)
			infoLog.Println("program_restart")
			time.Sleep(time.Minute)
			continue
		}
		db.SetMaxOpenConns(10)
		db.SetConnMaxIdleTime(time.Second * 10)
		db.SetConnMaxLifetime(time.Second * 10)

		pingErr := db.Ping()
		if pingErr != nil {
			errorLog.Println(err)
			infoLog.Println("program_restart")
			time.Sleep(time.Minute)
			continue
		}
		infoLog.Println("connected_db")
		defer db.Close()

		data_accounts, err := request_accounts(db)
		if err != nil {
			errorLog.Println(err)
			infoLog.Println("program_restart")
			time.Sleep(time.Minute)
			continue
		}
		infoLog.Println("request_accounts")
		infoLog.Println("activated_account:", data_accounts[active_account_id].Account_name)

		for {
			time_request := time.Now().Unix()

			data_task, err := request_last_task(db)
			infoLog.Println("request_last_task")
			if err != nil {
				errorLog.Println(err)
				infoLog.Println("program_restart")
				time.Sleep(time.Minute)
				_ = creation_new_task(db, time_request) //это вообще легально?

				break
			}

			if data_task.Date == 0 {
				infoLog.Println("no_last_task")

				creation_new_task(db, time_request)
				if err != nil {
					errorLog.Println(err)
					infoLog.Println("program_restart")
					time.Sleep(time.Minute)
					break
				}
				infoLog.Println("creation_new_task")
			}

			today := time.Unix(time_request, 0).Day()
			day_on_task := time.Unix(data_task.Date, 0).Day()

			if data_task.Completed == 1 { //завершено, смотрим на его дату,если:

				if today != day_on_task { //не сегодня, то создаем новое задание,
					infoLog.Println("last_task_completed_not_today")

					creation_new_task(db, time_request)
					if err != nil {
						errorLog.Println(err)
						infoLog.Println("program_restart")
						time.Sleep(time.Minute)
						break
					}
					infoLog.Println("creation_new_task")

					data_keys, err = request_keywords(db, 0)
					if err != nil {
						errorLog.Println(err)
						infoLog.Println("program_restart")
						time.Sleep(time.Minute)
						break
					}
					infoLog.Println("request_keywords(0)")

				} else { //сегодня, сделать паузу до конца суток
					infoLog.Println("last_task_completed_today")

					infoLog.Println("sleep_30_minutes") //////
					time.Sleep(time.Minute * 30)                                        //////

					continue
				}

			} else {
				infoLog.Println("last_task_not_completed")

				time_request = data_task.Date
				infoLog.Println("copy_time_request_from_task")

				if data_task.Primary_check == 0 {
					infoLog.Println("primary_check_not_completed")

					data_keys, err = request_keywords(db, data_task.Last_processed_key_id)
					if err != nil {
						errorLog.Println(err)
						infoLog.Println("program_restart")
						time.Sleep(time.Minute)
						break
					}
					infoLog.Printf(time.Now().Format(time.RFC822)+" request_keywords(%v)\n", data_task.Last_processed_key_id)

				} else {
					infoLog.Println("primary_check_completed")

					data_keys, err = request_statistics_zero_positions(db, time_request, max_test)
					if err != nil {
						errorLog.Println(err)
						infoLog.Println("program_restart")
						time.Sleep(time.Minute)
						break
					}
					infoLog.Println("request_statistics_zero_positions")

					if len(data_keys) < 1 {
						infoLog.Println("all_zero_positions_processed")

						insert, err := db.Query(`UPDATE task SET completed = 1 WHERE date = ?;`, time_request)
						if err != nil {
							errorLog.Println(err)
							infoLog.Println("program_restart")
							time.Sleep(time.Minute)
							break
						}
						insert.Close()
						infoLog.Println("task_completion")

						continue
					}
				}
			}

			infoLog.Println("keywords_processing...")

			error_flag := false

			for _, data := range data_keys {

				var url = fmt.Sprintf("https://yandex.ru/search/xml?user=%v&key=%v&lr=2&query=%v&groupby=",
					data_accounts[active_account_id].Account_name, data_accounts[active_account_id].Account_key,
					url.QueryEscape(data.Keyword_name)) /////

				respData, err := GetRequest(url+"groups-on-page%3D100", active_accout) ////
				if err != nil {
					errorLog.Println(err)
					infoLog.Println("program_restart")
					time.Sleep(time.Minute)
					break
				}

				v := models.Result{}

				xml.Unmarshal(respData, &v)
				infoLog.Println("GetRequest ", data.Keyword_name)

				response := v.Response

				if response.Error.Error_code != 0 {
					infoLog.Println("error code = ", response.Error.Error_code)
					//https://yandex.ru/dev/xml/doc/dg/reference/error-codes.html#error-codes

					active_account_id, active_accout, err = account_change(active_account_id, data_accounts)
					if err != nil {
						errorLog.Println(err)
						infoLog.Println("program_restart")
						time.Sleep(time.Minute)
						break
					}
					infoLog.Println("activated_account:", data_accounts[active_account_id].Account_name)
					infoLog.Println("account_change")

					if active_account_id == 0 {
						data_accounts, err = request_accounts(db)
						if err != nil {
							errorLog.Println(err)
							infoLog.Println("program_restart")
							time.Sleep(time.Minute)
							break
						}
						infoLog.Println("sleep_5_minutes")
						time.Sleep(time.Minute * 5)
						infoLog.Println("request_accounts")
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
							infoLog.Printf(time.Now().Format(time.RFC822)+" keyword_found_at_position_%v,_write_to_the_database\n", i)

							insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
						VALUES (?, ?, ?, ?, ?);`, i+1, vol.Url, time_request, data.Host_id, data.Keyword_id)
							if err != nil {
								errorLog.Println(err)
								infoLog.Println("program_restart")
								time.Sleep(time.Minute)
								break
							}
							insert.Close()

							insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.Keyword_id, time_request)
							if err != nil {
								errorLog.Println(err)
								infoLog.Println("program_restart")
								time.Sleep(time.Minute)
								break
							}
							insert.Close()

						} else { //старое значения в базе
							infoLog.Printf(time.Now().Format(time.RFC822)+" keyword_found_at_position_%v,_update_the_value_in_the_database\n", i)

							insert, err := db.Query(`UPDATE statistics SET position_num = ?, url = ?, test_number=test_number+1 
						WHERE keyword_id=? && date = ?;`, i+1, vol.Url, data.Keyword_id, time_request)
							if err != nil {
								errorLog.Println(err)
								infoLog.Println("program_restart")
								time.Sleep(time.Minute)
								break
							}
							insert.Close()
						}
						result_found = true
						break
					}
				}

				if !result_found {
					if data.In_statistics == 0 {
						infoLog.Println("keyword_not_found_write_position_zero_to_database")

						insert, err := db.Query(`INSERT INTO statistics (position_num, url, date, host_id, keyword_id) 
					VALUES (?, ?, ?, ?, ?)`, 0, vol.Url, time_request, data.Host_id, data.Keyword_id)
						if err != nil {
							errorLog.Println(err)
							infoLog.Println("program_restart")
							time.Sleep(time.Minute)
							break
						}
						insert.Close()
						insert, err = db.Query(`UPDATE task SET last_processed_key_id = ? WHERE date = ?;`, data.Keyword_id, time_request)
						if err != nil {
							errorLog.Println(err)
							infoLog.Println("program_restart")
							time.Sleep(time.Minute)
							break
						}
						insert.Close()
					} else {
						infoLog.Println("keyword_not_found_updating_test_information")

						insert, err := db.Query(`UPDATE statistics SET test_number=test_number+1 WHERE keyword_id=? && date = ?;`, data.Keyword_id, time_request)
						if err != nil {
							errorLog.Println(err)
							infoLog.Println("program_restart")
							time.Sleep(time.Minute)
							break
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
					errorLog.Println(err)
					infoLog.Println("program_restart")
					time.Sleep(time.Minute)
					break
				}
				insert.Close()
				infoLog.Println("primary_check_completed")
			}

		}
	}

}

func GetRequest(url string, active_accout *http.Client) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := active_accout.Do(req)
	if err != nil {
		return nil, err
	}
	respData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return respData, nil
}

func request_last_task(db *sql.DB) (models.Task, error) {
	row := db.QueryRow(`SELECT date, last_processed_key_id, primary_check, completed 
	FROM task ORDER BY id DESC LIMIT 1;`)

	var data_task models.Task

	if err := row.Scan(&data_task.Date, &data_task.Last_processed_key_id, &data_task.Primary_check,
		&data_task.Completed); err != nil {
		return data_task, err //завязано на логику
	}

	if err := row.Err(); err != nil {
		return data_task, err
	}

	return data_task, nil
}

func request_keywords(db *sql.DB, n int) ([]models.Keywords, error) {
	var data_keys []models.Keywords
	rows, err := db.Query(`SELECT keywords.id, keywords.keyword_name, hosts.id,hosts.host_name FROM keywords 
	LEFT JOIN hosts ON hosts.id=keywords.host_id WHERE keywords.id > ?;`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Keywords
		if err := rows.Scan(&data.Keyword_id, &data.Keyword_name, &data.Host_id, &data.Host_name); err != nil {
			return nil, err
		}
		data_keys = append(data_keys, data)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Keywords
		if err := rows.Scan(&data.Keyword_id, &data.Keyword_name, &data.Host_id, &data.Host_name); err != nil {
			return nil, err
		}
		data.In_statistics = 1
		data_keys = append(data_keys, data)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return data_keys, nil
}

func creation_new_task(db *sql.DB, time_request int64) error {
	insert, err := db.Query(`INSERT INTO task (date) VALUES (?);`, time_request)
	if err != nil {
		return err
	}
	insert.Close()
	return nil
}

func account_change(active_account_id int, data_accounts []models.Account) (int, *http.Client, error) {
	if active_account_id+1 < len(data_accounts) {
		active_account_id++

		proxyUrl, err := url.Parse(fmt.Sprintf(`http://%v:%v@%v`, data_accounts[active_account_id].Proxy_login,
			data_accounts[active_account_id].Proxy_password, data_accounts[active_account_id].Proxy_ip))
		if err != nil {
			return -1, nil, err
		}

		myClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}}

		return active_account_id, myClient, nil
	}

	return 0, http.DefaultClient, nil
}

func request_accounts(db *sql.DB) ([]models.Account, error) {
	var data_accounts []models.Account
	rows, err := db.Query(`SELECT * FROM accounts;`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var data models.Account
		if err := rows.Scan(&data.Id, &data.Account_name, &data.Account_key, &data.Proxy_ip,
			&data.Proxy_login, &data.Proxy_password); err != nil {
			return nil, err
		}
		data_accounts = append(data_accounts, data)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return data_accounts, nil
}
