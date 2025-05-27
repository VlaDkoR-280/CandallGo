package db

import (
	"container/list"
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

type DB struct {
	conn *pgxpool.Pool
}

func Connect(database_url string) (*DB, error) {
	pool, err0 := pgxpool.New(context.Background(), database_url)
	if err0 != nil {
		return nil, err0
	}
	//conn, err := pgx.Connect(context.Background(), database_url)
	//if err != nil {
	//	return nil, err
	//}
	//defer conn.Close(context.Background())

	log.Println("Connected to database")
	return &DB{conn: pool}, nil
}

func (db *DB) Close() {
	db.conn.Close()
}

func (db *DB) CheckExist(user_id string, group_id string, group_name string) error {
	var groupName string
	err := db.conn.QueryRow(context.Background(), "SELECT name FROM group_data WHERE Tg_id=$1", group_id).Scan(&groupName)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = db.conn.Exec(context.Background(), "INSERT INTO group_data (Tg_id, name) VALUES ($1, $2)", group_id, groupName)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		if groupName != group_name {
			_, err = db.conn.Exec(context.Background(), "UPDATE group_data SET name=$1 WHERE Tg_id=$2", group_name, group_id)
			if err != nil {
				return err
			}
		}
	}

	err = db.CheckExistUser(user_id, group_id)
	return err

	// Проверка наличия пользователя
	//var exist int
	//err := db.conn.QueryRow(context.Background(), "SELECT 1 FROM user_data WHERE Tg_id=$1", user_id).Scan(&exist)
	//if errors.Is(err, pgx.ErrNoRows) {
	//	_, err := db.conn.Exec(context.Background(), "INSERT INTO user_data (Tg_id) VALUES ($1)", user_id)
	//	if err != nil {
	//		return err
	//	}
	//} else if err != nil {
	//	return err
	//}
	//
	//// Проверка наличия группы
	//var groupName string
	//err = db.conn.QueryRow(context.Background(), "SELECT name FROM group_data WHERE Tg_id=$1", group_id).Scan(&groupName)
	//if errors.Is(err, pgx.ErrNoRows) {
	//	_, err := db.conn.Exec(context.Background(), "INSERT INTO group_data (Tg_id, name) VALUES ($1, $2)", group_id, group_name)
	//	if err != nil {
	//		return err
	//	}
	//} else if err != nil {
	//	return err
	//}
	//if groupName != group_name {
	//	_, err := db.conn.Exec(context.Background(), "UPDATE group_data SET name=$1 WHERE Tg_id=$2", group_name, group_id)
	//	if err != nil {
	//		return err
	//	}
	//}
	//
	//// Проверка наличия человека в группе
	//log.Println("SELECT 1 FROM users_of_group JOIN user_data u ON users_of_group.user_id=u.Id JOIN group_data g ON users_of_group.group_id=g.Id WHERE g.Tg_id=$1 AND u.Tg_id=$2", group_id, user_id)
	//err = db.conn.QueryRow(context.Background(), "SELECT 1 FROM users_of_group JOIN user_data u ON users_of_group.user_id=u.Id JOIN group_data g ON users_of_group.group_id=g.Id WHERE g.Tg_id=$1 AND u.Tg_id=$2", group_id, user_id).Scan(&exist)
	//if errors.Is(err, pgx.ErrNoRows) {
	//	var dbUserId int
	//	var dbGroupId int
	//	err = db.conn.QueryRow(context.Background(), "SELECT Id FROM user_data WHERE Tg_id=$1", user_id).Scan(&dbUserId)
	//	if err != nil {
	//		return err
	//	}
	//	err = db.conn.QueryRow(context.Background(), "SELECT Id FROM group_data WHERE Tg_id=$1", group_id).Scan(&dbGroupId)
	//
	//	_, err = db.conn.Exec(context.Background(), "INSERT INTO users_of_group (user_id, group_id) VALUES ($1, $2)", dbUserId, dbGroupId)
	//	if err != nil {
	//		return err
	//	}
	//}
	//
	//return nil
}

func (db *DB) CheckExistUser(user_id string, group_id string) error {
	var exist int
	err := db.conn.QueryRow(context.Background(), "SELECT 1 FROM user_data WHERE Tg_id=$1", user_id).Scan(&exist)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = db.conn.Exec(context.Background(), "INSERT INTO user_data (Tg_id) VALUES ($1)", user_id)
		if err != nil {
			return err
		}
		_, err = db.conn.Exec(context.Background(), "INSERT INTO users_of_group (user_id, group_id) SELECT u.Id, g.Id FROM user_data u JOIN group_data.g ON g.Tg_id=$1 WHERE u.Tg_id=$2", group_id, user_id)
		return err

	} else if err != nil {
		return err
	}
	err = db.conn.QueryRow(context.Background(), "SELECT 1 FROM users_of_group ug JOIN user_data u ON u.Id=ug.user_id JOIN group_data g ON g.Id=ug.group_id WHERE u.Tg_id=$1 AND g.Tg_id=$2", user_id, group_id).Scan(&exist)

	if errors.Is(err, pgx.ErrNoRows) {
		_, err = db.conn.Exec(context.Background(), "INSERT INTO users_of_group (user_id, group_id) SELECT u.Id, g.Id FROM user_data u JOIN group_data g ON g.Tg_id=$1 WHERE u.Tg_id=$2", group_id, user_id)
		return err
	}
	return err

}

func (db *DB) LeftUserFromGroup(user_id string, group_id string) error {
	userId, groupId := db.getLocalID(user_id, group_id)
	_, err := db.conn.Exec(context.Background(), "DELETE FROM users_of_group WHERE user_id=$1 AND group_id=$2", userId, groupId)
	return err
}

func (db *DB) getLocalID(user_id string, group_id string) (int, int) {
	var userID int
	var groupID int
	err := db.conn.QueryRow(context.Background(), "SELECT u.Id, g.Id FROM user_data u JOIN group_data g ON g.Tg_id=$1 WHERE u.Tg_id=$2", group_id, user_id).Scan(&userID, &groupID)
	if err != nil {
		return -1, -1
	}
	return userID, groupID
}
func (db *DB) CheckTimeUpdate(groupId string) (bool, error) {
	var timeUpdate time.Time
	err := db.conn.QueryRow(context.Background(), "SELECT time_last_update FROM group_data WHERE Tg_id=$1", groupId).Scan(&timeUpdate)
	return time.Since(timeUpdate).Minutes() >= 20, err
}

func (db *DB) SetTimeUpdate(group_id string) error {
	_, err := db.conn.Exec(context.Background(), "UPDATE group_data SET time_last_update = now() WHERE Tg_id=$1", group_id)
	return err
}

func (db *DB) GetUsersOfGroup(user_id string, group_id string, listUser *list.List) error {
	_, group_id_int := db.getLocalID(user_id, group_id)
	rows, err := db.conn.Query(context.Background(), "SELECT u.Tg_id FROM user_data u JOIN users_of_group ug ON ug.user_id=u.Id WHERE ug.group_id=$1", group_id_int)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var tg_id string
		err = rows.Scan(&tg_id)
		if err != nil {
			return err
		}
		if tg_id != user_id {
			listUser.PushBack(tg_id)
		}

	}
	return nil
}

type GroupData struct {
	Tg_id string
	Name  string
	Id    int64
}

func (db *DB) GetGroupsOfUser(user_id string, group_id string, listGroup *list.List) error {

	var userIdInt int64
	err := db.conn.QueryRow(context.Background(), "SELECT id FROM user_data WHERE tg_id=$1", user_id).Scan(&userIdInt)
	if errors.Is(err, pgx.ErrNoRows) {
		userIdInt = -1
	} else if err != nil {
		return err
	}
	//err = db.conn.QueryRow(context.Background(), "SELECT id FROM group_data WHERE tg_id=$1", group_id).Scan(&groupIdInt)
	//if errors.Is(err, pgx.ErrNoRows) {
	//	groupIdInt = -1
	//} else if err != nil {
	//	return err
	//}
	rows, err := db.conn.Query(context.Background(), "SELECT Id, Tg_id, name FROM group_data g JOIN users_of_group ug ON ug.group_id=g.id WHERE ug.user_id=$1", userIdInt)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var groupData GroupData
		err = rows.Scan(&groupData.Id, &groupData.Tg_id, &groupData.Name)
		if err != nil {
			return err
		}
		listGroup.PushBack(groupData)
	}
	return nil
}

func (db *DB) GetSubEndDate(group_id string, date *time.Time) error {
	err := db.conn.QueryRow(context.Background(), "SELECT sub_end_date FROM group_data WHERE Tg_id=$1", group_id).Scan(date)
	return err
}

func (db *DB) GetLastTimeUse(groupId string, timeLastUse *time.Time) error {

	err := db.conn.QueryRow(context.Background(), "SELECT data_last_use FROM group_data WHERE Tg_id=$1", groupId).Scan(timeLastUse)
	if timeLastUse.Truncate(24 * time.Hour).Equal(time.Now().Truncate(24 * time.Hour)) {
		return err
	}
	_, err = db.conn.Exec(context.Background(), "UPDATE group_data SET data_last_use=now() WHERE Tg_id=$1 ", groupId)
	return err
}
