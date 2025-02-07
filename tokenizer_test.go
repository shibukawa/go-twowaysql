package twowaysql

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []token
	}{
		{
			name:  "empty",
			input: "",
			want: []token{
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "no comment",
			input: "SELECT * FROM person WHERE employee_no < 1000  AND dept_no = 1",
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "SELECT * FROM person WHERE employee_no < 1000  AND dept_no = 1",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "bind  space 1",
			input: `SELECT * FROM person WHERE first_name = /* firstName */"Jeff Dean"`,
			want: []token{
				{
					kind: tkSQLStmt,
					str:  `SELECT * FROM person WHERE first_name = `,
				},
				{
					kind:  tkBind,
					str:   "?/* firstName */",
					value: "firstName",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "bind  space 2",
			input: `SELECT * FROM person WHERE first_name = /* firstName */'Jeff Dean'`,
			want: []token{
				{
					kind: tkSQLStmt,
					str:  `SELECT * FROM person WHERE first_name = `,
				},
				{
					kind:  tkBind,
					str:   "?/* firstName */",
					value: "firstName",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "bind  space 3",
			input: `SELECT * FROM person WHERE first_name = /* firstName */"Jeff Dean" AND deptNo < 10`,
			want: []token{
				{
					kind: tkSQLStmt,
					str:  `SELECT * FROM person WHERE first_name = `,
				},
				{
					kind:  tkBind,
					str:   "?/* firstName */",
					value: "firstName",
				},
				{
					kind: tkSQLStmt,
					str:  ` AND deptNo < 10`,
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "insert",
			input: `INSERT INTO persons (employee_no, dept_no, first_name, last_name, email) VALUES(/*EmpNo*/1, /*deptNo*/1)`,
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "INSERT INTO persons (employee_no, dept_no, first_name, last_name, email) VALUES(",
				},
				{
					kind:  tkBind,
					str:   "?/*EmpNo*/",
					value: "EmpNo",
				},
				{
					kind: tkSQLStmt,
					str:  ", ",
				},
				{
					kind:  tkBind,
					str:   "?/*deptNo*/",
					value: "deptNo",
				},
				{
					kind: tkSQLStmt,
					str:  ")",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "if",
			input: "SELECT * FROM person WHERE employee_no < 1000 /* IF true */ AND dept_no = 1/* END */",
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "SELECT * FROM person WHERE employee_no < 1000 ",
				},
				{
					kind:      tkIf,
					str:       "/* IF true */",
					condition: "true",
				},
				{
					kind: tkSQLStmt,
					str:  " AND dept_no = 1",
				},
				{
					kind: tkEnd,
					str:  "/* END */",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "if and bind",
			input: "SELECT * FROM person WHERE employee_no < /*maxEmpNo*/1000 /* IF false */ AND dept_no = /*deptNo*/1 /* END */",
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "SELECT * FROM person WHERE employee_no < ",
				},
				{
					kind:  tkBind,
					str:   "?/*maxEmpNo*/",
					value: "maxEmpNo",
				},
				{
					kind: tkSQLStmt,
					str:  " ",
				},
				{
					kind:      tkIf,
					str:       "/* IF false */",
					condition: "false",
				},
				{
					kind: tkSQLStmt,
					str:  " AND dept_no = ",
				},
				{
					kind:  tkBind,
					str:   "?/*deptNo*/",
					value: "deptNo",
				},
				{
					kind: tkSQLStmt,
					str:  " ",
				},
				{
					kind: tkEnd,
					str:  "/* END */",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "if elif else",
			input: "SELECT * FROM person WHERE employee_no < 1000 /* IF true */AND dept_no =1/* ELIF true*/ AND boss_no = 2 /*ELSE */ AND id=3/* END */",
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "SELECT * FROM person WHERE employee_no < 1000 ",
				},
				{
					kind:      tkIf,
					str:       "/* IF true */",
					condition: "true",
				},
				{
					kind: tkSQLStmt,
					str:  "AND dept_no =1",
				},
				{
					kind:      tkElif,
					str:       "/* ELIF true*/",
					condition: "true",
				},
				{
					kind: tkSQLStmt,
					str:  " AND boss_no = 2 ",
				},
				{
					kind: tkElse,
					str:  "/*ELSE */",
				},
				{
					kind: tkSQLStmt,
					str:  " AND id=3",
				},
				{
					kind: tkEnd,
					str:  "/* END */",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "if nest",
			input: "SELECT * FROM person WHERE employee_no < 1000 /* IF true */ /* IF false */ AND dept_no =1 /* ELSE */ AND id=3 /* END */ /* ELSE*/ AND boss_id=4 /* END */",
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "SELECT * FROM person WHERE employee_no < 1000 ",
				},
				{
					kind:      tkIf,
					str:       "/* IF true */",
					condition: "true",
				},
				{
					kind: tkSQLStmt,
					str:  " ",
				},
				{
					kind:      tkIf,
					str:       "/* IF false */",
					condition: "false",
				},
				{
					kind: tkSQLStmt,
					str:  " AND dept_no =1 ",
				},
				{
					kind: tkElse,
					str:  "/* ELSE */",
				},
				{
					kind: tkSQLStmt,
					str:  " AND id=3 ",
				},
				{
					kind: tkEnd,
					str:  "/* END */",
				},
				{
					kind: tkSQLStmt,
					str:  " ",
				},
				{
					kind: tkElse,
					str:  "/* ELSE*/",
				},
				{
					kind: tkSQLStmt,
					str:  " AND boss_id=4 ",
				},
				{
					kind: tkEnd,
					str:  "/* END */",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
		{
			name:  "in bind",
			input: `SELECT * FROM person /* IF gender_list !== null */ WHERE person.gender in /*gender_list*/('M') /* END */`,
			want: []token{
				{
					kind: tkSQLStmt,
					str:  "SELECT * FROM person ",
				},
				{
					kind:      tkIf,
					str:       "/* IF gender_list !== null */",
					condition: "gender_list !== null",
				},
				{
					kind: tkSQLStmt,
					str:  " WHERE person.gender in ",
				},
				{
					kind:  tkBind,
					str:   "?/*gender_list*/",
					value: "gender_list",
				},
				{
					kind: tkSQLStmt,
					str:  " ",
				},
				{
					kind: tkEnd,
					str:  "/* END */",
				},
				{
					kind: tkEndOfProgram,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := tokenize(tt.input); err != nil || !tokensEqual(tt.want, got) {
				if err != nil {
					t.Error(err)
				}
				t.Errorf("Doesn't Match expected: %v, but got: %v\n", tt.want, got)
			}
		})
	}
}
func TestTokenizeShouldReturnError(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError string
	}{
		{
			name:      "bad comment format",
			input:     "SELECT * FROM person WHERE employee_no < 1000 /* IF true / AND dept_no = 1",
			wantError: "Comment enclosing characters do not match",
		},
		{
			name:      "Enclosing characters not match 1",
			input:     `SELECT * FROM person WHERE employee_no < /* firstName */"Jeff Dean AND dept_no = 1`,
			wantError: "Enclosing characters do not match",
		},
		{
			name:      "Enclosing characters not match 2",
			input:     `SELECT * FROM person WHERE employee_no < /* firstName */"Jeff Dean' AND dept_no = 1`,
			wantError: "Enclosing characters do not match",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := tokenize(tt.input); err == nil || err.Error() != tt.wantError {
				if err == nil {
					t.Error("Should Error")
				} else if err.Error() != tt.wantError {
					t.Errorf("Doesn't Match expected: %v, but got: %v\n", tt.wantError, err.Error())
				}
			}
		})
	}
}

func tokensEqual(want, got []token) bool {
	if len(want) != len(got) {
		return false
	}
	for i := 0; i < len(want); i++ {
		if want[i] != got[i] {
			return false
		}
	}
	return true
}
