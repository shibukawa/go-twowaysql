package twowaysql

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Person struct {
	FirstName string `db:"first_name"`
	LastName  string `db:"last_name"`
	Email     string `db:"email"`
}

func TestSelect(t *testing.T) {
	//このテストはinit.sqlに依存しています。
	//データベースは/postgres/init以下のsqlファイルを用いて初期化されている。
	db := open(t)
	defer db.Close()
	tw := New(db)
	ctx := context.Background()

	// SELECT
	var people []Person
	var params = Info{
		MaxEmpNo: 3,
		DeptNo:   12,
	}

	expected := []Person{
		{
			FirstName: "Evan",
			LastName:  "MacMans",
			Email:     "evanmacmans@example.com",
		},
		{
			FirstName: "Malvina",
			LastName:  "FitzSimons",
			Email:     "malvinafitzsimons@example.com",
		},
	}

	err := tw.Select(ctx, &people, `SELECT first_name, last_name, email FROM persons WHERE employee_no < /*maxEmpNo*/1000 /* IF deptNo */ AND dept_no < /*deptNo*/1 /* END */`, &params)
	if err != nil {
		t.Errorf("select: failed: %v", err)
	}

	if !match(people, expected) {
		t.Errorf("\nexpected:\n%v\nbut got\n%v\n", expected, people)
	}

}

func TestUpdate(t *testing.T) {
	//このテストはinit.sqlに依存しています。
	//データベースは/postgres/init以下のsqlファイルを用いて初期化されている。
	db := open(t)
	defer db.Close()
	tw := New(db)
	ctx := context.Background()

	var params = Info{
		MaxEmpNo: 3,
		DeptNo:   12,
	}
	_, err := tw.Exec(ctx, `UPDATE persons SET dept_no = /*deptNo*/1 WHERE employee_no = /*EmpNo*/1`, &params)
	if err != nil {
		t.Fatalf("exec: failed: %v", err)
	}
	var people []Person
	err = tw.Select(ctx, &people, `SELECT first_name, last_name, email FROM persons WHERE dept_no = 11`, nil)
	if err != nil {
		t.Fatalf("select: failed: %v", err)
	}
	// 元に戻す。本当はトランザクションのラッパーを実装するべきかも
	_, err = tw.Exec(ctx, `UPDATE persons SET dept_no = /*deptNo*/0 WHERE employee_no = /*EmpNo*/1`, &params)
	if err != nil {
		t.Fatalf("exec: failed: %v", err)
	}
	var expected = []Person{
		{
			FirstName: "Malvina",
			LastName:  "FitzSimons",
			Email:     "malvinafitzsimons@example.com",
		},
	}
	if !match(people, expected) {
		t.Errorf("expected:\n%v\nbut got\n%v\n", expected, people)
	}
}

func TestInsertAndDelete(t *testing.T) {
	//このテストはinit.sqlに依存しています。
	//データベースは/postgres/init以下のsqlファイルを用いて初期化されている。
	db := open(t)
	defer db.Close()
	tw := New(db)
	ctx := context.Background()

	var params = Info{
		EmpNo:     100,
		FirstName: "Jeff",
		LastName:  "Dean",
		DeptNo:    1011,
		Email:     "jeffdean@example.com",
	}
	_, err := tw.Exec(ctx, `INSERT INTO persons (employee_no, dept_no, first_name, last_name, email) VALUES(/*EmpNo*/1, /*deptNo*/1, /*firstName*/"Tim", /*lastName*/"Cook", /*email*/"timcook@example.com")`, &params)
	if err != nil {
		t.Fatalf("exec: failed: %v", err)
	}

	var people []Person
	err = tw.Select(ctx, &people, `SELECT first_name, last_name, email FROM persons WHERE dept_no = /*deptNo*/0`, &params)
	if err != nil {
		t.Fatalf("select: failed: %v", err)
	}

	var expected = []Person{
		{
			FirstName: "Jeff",
			LastName:  "Dean",
			Email:     "jeffdean@example.com",
		},
	}
	if !match(people, expected) {
		t.Errorf("expected:\n%v\nbut got\n%v\n", expected, people)
	}

	_, err = tw.Exec(ctx, `DELETE FROM persons WHERE employee_no = /*EmpNo*/2`, &params)
	if err != nil {
		t.Fatalf("exec: failed: %v", err)
	}

	people = []Person{}
	err = tw.Select(ctx, &people, `SELECT first_name, last_name, email FROM persons WHERE dept_no = /*deptNo*/0`, &params)
	if err != nil {
		t.Fatalf("select: failed: %v", err)
	}

	expected = []Person{}
	if !match(people, expected) {
		t.Errorf("expected:\n%v\nbut got\n%v\n", expected, people)
	}
}

func TestTxCommit(t *testing.T) {
	//このテストはinit.sqlに依存しています。
	//データベースは/postgres/init以下のsqlファイルを用いて初期化されている。
	db := open(t)
	defer db.Close()
	tw := New(db)
	ctx := context.Background()

	// insert test data
	const insertSQL = `
	INSERT INTO persons
		(employee_no, dept_no, first_name, last_name, email) VALUES
		(11, 111, 'Clegg', 'George', 'clegggeorge@example.com')
		;
	`
	if _, err := tw.Exec(ctx, insertSQL, nil); err != nil {
		t.Fatal(err)
	}
	defer tw.Exec(ctx, `DELETE FROM persons WHERE employee_no = 11`, nil)

	// begin
	tx, err := tw.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// update
	type Param struct {
		EmpNo     int    `twowaysql:"EmpNo"`
		FirstName string `twowaysql:"FirstName"`
	}
	const sql = `
	UPDATE
		persons
	SET first_name = /*FirstName*/Jon
	WHERE employee_no = /*EmpNo*/10`
	param := Param{EmpNo: 11, FirstName: "Rimmer"}
	res, err := tx.Exec(ctx, sql, &param)
	if err != nil {
		t.Error(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		t.Error(err)
	}
	if rows != 1 {
		t.Errorf("update rows = %v", rows)
	}

	// commit
	if err := tx.Commit(); err != nil {
		t.Error(err)
	}

	// check
	people := []Person{}
	if err := tw.Select(ctx, &people, `SELECT first_name, last_name, email FROM persons WHERE employee_no = /*EmpNo*/10`, &param); err != nil {
		t.Error(err)
	}
	expectedAfterCommit := []Person{
		{
			FirstName: "Rimmer",
			LastName:  "George",
			Email:     "clegggeorge@example.com",
		},
	}
	if !match(expectedAfterCommit, people) {
		t.Errorf("expected:\n%v\nbut got\n%v\n", expectedAfterCommit, people)
	}
}

func TestTxRollback(t *testing.T) {
	//このテストはinit.sqlに依存しています。
	//データベースは/postgres/init以下のsqlファイルを用いて初期化されている。
	db := open(t)
	defer db.Close()
	tw := New(db)
	ctx := context.Background()

	// insert test data
	const insertSQL = `
	INSERT INTO persons
		(employee_no, dept_no, first_name, last_name, email) VALUES
		(12, 121, 'Chmmg', 'Dudley', 'chmmgdudley@example.com')
		;
	`
	if _, err := tw.Exec(ctx, insertSQL, nil); err != nil {
		t.Fatal(err)
	}
	defer tw.Exec(ctx, `DELETE FROM persons WHERE employee_no = 12`, nil)

	// begin
	tx, err := tw.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// update
	type Param struct {
		EmpNo     int    `twowaysql:"EmpNo"`
		FirstName string `twowaysql:"firstName"`
	}
	const sql = `
	UPDATE
		persons
	SET first_name = /*firstName*/Jon
	WHERE employee_no = /*EmpNo*/10`
	param := Param{EmpNo: 12, FirstName: "Emerson"}
	res, err := tx.Exec(ctx, sql, &param)
	if err != nil {
		t.Error(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		t.Error(err)
	}
	if rows != 1 {
		t.Errorf("update rows = %v", rows)
	}

	// rollback
	if err := tx.Rollback(); err != nil {
		t.Error(err)
	}

	// check
	people := []Person{}
	if err := tw.Select(ctx, &people, `SELECT first_name, last_name, email FROM persons WHERE employee_no = /*EmpNo*/10`, &param); err != nil {
		t.Error(err)
	}
	expectedAfterCommit := []Person{
		{
			FirstName: "Chmmg",
			LastName:  "Dudley",
			Email:     "chmmgdudley@example.com",
		},
	}
	if !match(expectedAfterCommit, people) {
		t.Errorf("expected:\n%v\nbut got\n%v\n", expectedAfterCommit, people)
	}
}

func TestTxBlock(t *testing.T) {
	//このテストはinit.sqlに依存しています。
	//データベースは/postgres/init以下のsqlファイルを用いて初期化されている。
	db := open(t)
	defer db.Close()
	tw := New(db)
	ctx := context.Background()

	// insert test data
	const insertSQL = `
	INSERT INTO persons
		(employee_no, dept_no, first_name, last_name, email) VALUES
		(13, 131, 'Darling', 'Wat', 'darlingwat@example.com'),
		(14, 141, 'Hallows', 'Jessie', 'hallowsjessie@example.com')
		;`
	if _, err := tw.Exec(ctx, insertSQL, nil); err != nil {
		t.Fatal(err)
	}
	defer tw.Exec(ctx, `DELETE FROM persons WHERE employee_no = 13`, nil)
	defer tw.Exec(ctx, `DELETE FROM persons WHERE employee_no = 14`, nil)

	type Param struct {
		EmpNo     int    `twowaysql:"EmpNo"`
		FirstName string `twowaysql:"firstName"`
	}
	// commit case
	err := tw.Transaction(ctx, func(tx TwowaysqlTx) error {
		// update
		const sql = `
		UPDATE
			persons
		SET first_name = /*firstName*/Jon
		WHERE employee_no = /*EmpNo*/10`
		param := Param{EmpNo: 13, FirstName: "COMMITED"}
		res, err := tx.Exec(ctx, sql, &param)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("update rows = %v", rows)
		}

		return nil
	})
	if err != nil {
		t.Error(err)
	}

	// rollcack case
	err = tw.Transaction(ctx, func(tx TwowaysqlTx) error {
		// update
		const sql = `
		UPDATE
			persons
		SET first_name = /*firstName*/Jon
		WHERE employee_no = /*EmpNo*/10`
		param := Param{EmpNo: 14, FirstName: "ROLLBACKED"}
		res, err := tx.Exec(ctx, sql, &param)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows != 1 {
			return fmt.Errorf("update rows = %v", rows)
		}

		// generate error
		return errors.New("TEST ERROR")
	})
	if err == nil {
		t.Error("unexpecte err == nil")
	}

	// check
	people := []Person{}
	const checkSQL = `SELECT first_name, last_name, email FROM persons WHERE employee_no IN (13, 14) order by employee_no`
	if err := tw.Select(ctx, &people, checkSQL, nil); err != nil {
		t.Error(err)
	}
	expectedAfterCommit := []Person{
		// commit
		{
			FirstName: "COMMITED",
			LastName:  "Wat",
			Email:     "darlingwat@example.com",
		},
		// rollback
		{
			FirstName: "Hallows",
			LastName:  "Jessie",
			Email:     "hallowsjessie@example.com",
		},
	}
	if !match(expectedAfterCommit, people) {
		t.Errorf("expected:\n%v\nbut got\n%v\n", expectedAfterCommit, people)
	}

}

func open(t *testing.T) *sqlx.DB {
	var db *sqlx.DB
	var err error

	if host := os.Getenv("POSTGRES_HOST"); host != "" {
		db, err = sqlx.Open("postgres", fmt.Sprintf("host=%s user=postgres password=postgres dbname=postgres sslmode=disable", host))
	} else {
		db, err = sqlx.Open("postgres", "user=postgres password=postgres dbname=postgres sslmode=disable")
	}

	if err != nil {
		t.Fatal(err)
	}

	return db
}

func match(p1, p2 []Person) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i := 0; i < len(p1); i++ {
		if p1[i] != p2[i] {
			return false
		}
	}
	return true
}
