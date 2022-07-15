#!/bin/bash

/etc/init.d/mysql stop
/usr/bin/mysql_install_db
/etc/init.d/mysql start
/usr/bin/mysqladmin -u root password $MYSQL_USER_NAME_PASS
mysql -u root -p$MYSQL_USER_NAME_PASS -e "CREATE DATABASE $MYSQL_DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
mysql -u root -p$MYSQL_USER_NAME_PASS -e "CREATE USER '$MYSQL_USER_NAME'@'%' IDENTIFIED BY '$MYSQL_USER_NAME_PASS';"
mysql -u root -p$MYSQL_USER_NAME_PASS -e "GRANT ALL PRIVILEGES ON $MYSQL_DB_NAME . * TO '$MYSQL_USER_NAME'@'%';"
mysql -u root -p$MYSQL_USER_NAME_PASS -e "FLUSH PRIVILEGES"

mysql -u $MYSQL_USER_NAME -p$MYSQL_USER_NAME_PASS -e "CREATE TABLE $MYSQL_DB_NAME.accounts (id int(11) unsigned NOT NULL AUTO_INCREMENT,
account_name varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
account_key varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
proxy_ip varchar(25) COLLATE utf8mb4_unicode_ci DEFAULT '0',
proxy_login varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT '0',
proxy_password varchar(50) COLLATE utf8mb4_unicode_ci DEFAULT '0',
PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=3 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"

mysql -u $MYSQL_USER_NAME -p$MYSQL_USER_NAME_PASS -e "CREATE TABLE $MYSQL_DB_NAME.hosts (
id int(11) unsigned NOT NULL AUTO_INCREMENT,
host_name varchar(50) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"

mysql -u $MYSQL_USER_NAME -p$MYSQL_USER_NAME_PASS -e "CREATE TABLE $MYSQL_DB_NAME.keywords (
id int(11) unsigned NOT NULL AUTO_INCREMENT,
keyword_name varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
host_id int(11) unsigned DEFAULT NULL,
PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=884 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"

mysql -u $MYSQL_USER_NAME -p$MYSQL_USER_NAME_PASS -e "CREATE TABLE $MYSQL_DB_NAME.statistics (
id int(11) unsigned NOT NULL AUTO_INCREMENT,
position_num tinyint(4) unsigned DEFAULT NULL,
url varchar(256) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
date int(11) unsigned DEFAULT NULL,
host_id int(11) unsigned DEFAULT NULL,
keyword_id int(11) unsigned DEFAULT NULL,
test_number tinyint(4) unsigned DEFAULT '1',
PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=4795 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"

mysql -u $MYSQL_USER_NAME -p$MYSQL_USER_NAME_PASS -e "CREATE TABLE $MYSQL_DB_NAME.task (
id int(11) unsigned NOT NULL AUTO_INCREMENT,
date int(11) unsigned DEFAULT NULL,
last_processed_key_id int(11) unsigned DEFAULT '0',
primary_check tinyint(1) unsigned DEFAULT '0',
completed tinyint(1) unsigned DEFAULT '0',
PRIMARY KEY (id)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"