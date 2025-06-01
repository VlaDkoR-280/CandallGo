package db

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"strings"
	"time"
)

// AddUser добавление пользователя в БД
func (db *DB) AddUser(userId string) error {
	if userId == "" {
		return errors.New("userId is empty")
	}
	pgTag, err := db.conn.Exec(context.Background(),
		"INSERT INTO user_data (tg_id) VALUES ($1)", userId)
	if err != nil {
		return myErrorExec(err, pgTag)
	}
	return nil
}

// RemoveUser удаление пользователя из БД
func (db *DB) RemoveUser(userId string) error {
	return errors.New("Блок недописан")
}

// AddGroup добавление группы в БД
func (db *DB) AddGroup(groupId, name string, isGroup bool) error {
	pgTag, err := db.conn.Exec(context.Background(),
		"INSERT INTO group_data (tg_id, name, isgroup) VALUES "+
			"($1, $2, $3)",
		groupId, name, isGroup)
	if err != nil {
		return myErrorExec(err, pgTag)
	}
	return nil
}

// UpdateGroupData обновление данных о группе по GroupData, поле GroupId
// не должно быть пустым, остальные параметры, которые не нужно обновлять
// должны быть путсыми (nil, "", time.isZero())
func (db *DB) UpdateGroupData(newGroupData GroupData) error {
	query, queryParams, err := generateNeedUpdate(newGroupData)
	if err != nil {
		return err
	}
	pgTag, err := db.conn.Exec(context.Background(),
		query, queryParams...)
	if err != nil {
		return myErrorExec(err, pgTag)
	}
	return nil
}

// generateNeedUpdate создвние строки запроса  для pgx только с теми полями
// которые необходимо обновить, в формате:
// "UPDATE group_data SET " + "name=$1, ..." + WHERE tg_id=" + "$X", tg_id
func generateNeedUpdate(groupData GroupData) (query string, paramsValues []interface{}, err error) {
	var paramsString []string
	argId := 1
	if groupData.GroupName != "" {
		paramsString = append(paramsString, fmt.Sprintf("name=$%d", argId))
		paramsValues = append(paramsValues, groupData.GroupName)
		argId++
	}
	if !groupData.DateLastUse.IsZero() {
		paramsString = append(paramsString, fmt.Sprintf("date_last_use=$%d", argId))
		paramsValues = append(paramsValues, groupData.DateLastUse)
		argId++
	}
	if !groupData.SubDateEnd.IsZero() {
		paramsString = append(paramsString, fmt.Sprintf("sub_end_date=$%d", argId))
		paramsValues = append(paramsValues, groupData.SubDateEnd)
		argId++
	}
	if !groupData.TimeLastUpdate.IsZero() {
		paramsString = append(paramsString, fmt.Sprintf("time_last_update=$%d", argId))
		paramsValues = append(paramsValues, groupData.TimeLastUpdate)
		argId++
	}
	if len(paramsString) <= 0 {
		return "", paramsValues, errors.New("All params is empty")
	}
	query = fmt.Sprintf("UPDATE group_data SET %s WHERE tg_id=$%d", strings.Join(paramsString, ", "), argId)
	paramsValues = append(paramsValues, groupData.TgId)
	return query, paramsValues, nil
}

// myErrorExec вывод ошибки при запросе Exec
func myErrorExec(mErr error, pgTag pgconn.CommandTag) error {
	return errors.New(mErr.Error() + " | [PG_ANSWER]:: " + pgTag.String())
}

func (db *DB) AddUserToGroup(userId, groupId string) error {
	userData, err := db.GetUserData(userId)
	if err != nil {
		return err
	}
	groupData, err := db.GetGroupData(groupId)
	if err != nil {
		return err
	}
	pgTag, err := db.conn.Exec(context.Background(), "INSERT INTO users_of_group (user_id, group_id) VALUES ($1, $2)", userData.Id, groupData.Id)
	if err != nil {
		return myErrorExec(err, pgTag)
	}
	return nil
}

func (db *DB) UpdateSubDate(groupId string, dateTime time.Time) error {
	_, err := db.conn.Exec(context.Background(),
		"UPDATE group_data SET sub_end_date=$1 WHERE tg_id=$2", dateTime, groupId)
	return err
}
