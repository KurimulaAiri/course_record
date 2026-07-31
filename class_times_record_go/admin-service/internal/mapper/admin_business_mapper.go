// Package mapper admin-service 业务管理透传数据访问层
//
// 对齐 Java admin-service AdminBusinessServiceImpl 直接操作业务表（c_student, c_teacher 等）
//
// 注意：本 Mapper 与 business-service 的 Mapper 操作相同的业务表，但查询方法独立
//       （admin 端查询需要关联账号/家长等额外信息，且不分角色权限过滤）
//
// 涵盖模块：
//   - 机构管理（c_institution + c_subscription_plan）
//   - 学生管理（c_student + c_parent_student + c_parent）
//   - 教师管理（c_teacher + c_user_auth）
//   - 课程管理（c_course）
//   - 班级管理（c_class + c_class_student + c_class_teacher）
//   - 课表管理（c_class_schedule）
//   - 课时记录管理（c_course_record）
//   - 上课记录管理（c_record）
//   - 小程序菜单管理（c_menu + c_role_menu）
package mapper

import (
	"database/sql"
	"fmt"
	"strings"
)

// ============================================================
// 机构管理
// ============================================================

// AdminInstitutionRow 机构查询行（对齐 Java AdminInstitutionVO）
//
// 使用普通类型而非 sql.NullXxx，便于 JSON 序列化
type AdminInstitutionRow struct {
	ID                   int64  `json:"id"`                   // 机构ID
	InstitutionName      string `json:"institutionName"`      // 机构名称
	InstitutionAddress   string `json:"institutionAddress"`   // 机构地址
	InstitutionCode      string `json:"institutionCode"`      // 机构编码（用于登录）
	Status               int64  `json:"status"`               // 状态（0=待审核,1=启用,2=禁用）
	ExpireTimeStr        string `json:"expireTime"`           // 过期时间字符串（yyyy-MM-dd HH:mm:ss，空表示永久有效）
	SubscriptionPlanID   int64  `json:"subscriptionPlanId"`   // 订阅套餐ID
	SubscriptionPlanName string `json:"subscriptionPlanName"` // 订阅套餐名称（JOIN 获取）
	CreateTimeStr        string `json:"createTimeStr"`        // 创建时间字符串
	UpdateTimeStr        string `json:"updateTimeStr"`        // 更新时间字符串
}

// AdminBusinessMapper 业务管理透传 Mapper
//
// 聚合所有业务表的查询/写入操作，供 AdminBusinessService 调用
// 与 business-service 的 Mapper 操作相同的表，但查询方法独立（admin 端无角色过滤）
type AdminBusinessMapper struct {
	db *sql.DB
}

// NewAdminBusinessMapper 创建 AdminBusinessMapper
func NewAdminBusinessMapper(db *sql.DB) *AdminBusinessMapper {
	return &AdminBusinessMapper{db: db}
}

// scanInstitution 通用机构行扫描函数
//
// 复用 sql.Row 和 sql.Rows 的 Scan 方法
func scanInstitution(scanner interface {
	Scan(dest ...interface{}) error
}) (*AdminInstitutionRow, error) {
	row := &AdminInstitutionRow{}
	var (
		name, address, code, planName sql.NullString
		status, planID                sql.NullInt64
		expireTime, createTime, updateTime sql.NullTime
	)
	err := scanner.Scan(
		&row.ID, &name, &address, &code, &status, &expireTime, &planID, &planName, &createTime, &updateTime,
	)
	if err != nil {
		return nil, err
	}
	row.InstitutionName = name.String
	row.InstitutionAddress = address.String
	row.InstitutionCode = code.String
	if status.Valid {
		row.Status = status.Int64
	}
	row.ExpireTimeStr = formatTimeSQL(expireTime)
	if planID.Valid {
		row.SubscriptionPlanID = planID.Int64
	}
	row.SubscriptionPlanName = planName.String
	row.CreateTimeStr = formatTimeSQL(createTime)
	row.UpdateTimeStr = formatTimeSQL(updateTime)
	return row, nil
}

// ListInstitutions 机构分页列表（对齐 Java listInstitutions）
//
// 查询参数：
//   - institutionID: 机构ID精确匹配（0 不过滤）
//   - institutionName: 机构名称模糊匹配（空不过滤）
//   - institutionCode: 机构编码模糊匹配（空不过滤）
//   - status: 状态精确匹配（0 不过滤）
//   - offset/limit: 分页参数
//
// 返回：机构列表 + 总数
func (m *AdminBusinessMapper) ListInstitutions(institutionID int64, institutionName, institutionCode string, status int64, offset, limit int) ([]*AdminInstitutionRow, int64, error) {
	// 动态构造 WHERE 条件
	where := "WHERE 1=1"
	args := []interface{}{}
	if institutionID != 0 {
		where += " AND i.id = ?"
		args = append(args, institutionID)
	}
	if institutionName != "" {
		where += " AND i.institution_name LIKE ?"
		args = append(args, "%"+institutionName+"%")
	}
	if institutionCode != "" {
		where += " AND i.institution_code LIKE ?"
		args = append(args, "%"+institutionCode+"%")
	}
	if status != 0 {
		where += " AND i.status = ?"
		args = append(args, status)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM c_institution i %s`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计机构数失败: %w", err)
	}

	// 分页查询（LEFT JOIN c_subscription_plan 获取套餐名称）
	query := fmt.Sprintf(`
		SELECT i.id, i.institution_name, i.institution_address, i.institution_code,
		       i.status, i.expire_time, i.subscription_plan_id, p.plan_name,
		       i.create_time, i.update_time
		FROM c_institution i
		LEFT JOIN c_subscription_plan p ON i.subscription_plan_id = p.id
		%s
		ORDER BY i.create_time DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询机构列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminInstitutionRow
	for rows.Next() {
		row, err := scanInstitution(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("扫描机构记录失败: %w", err)
		}
		list = append(list, row)
	}
	return list, total, nil
}

// SelectInstitutionByID 按ID查机构（含订阅套餐名称）
func (m *AdminBusinessMapper) SelectInstitutionByID(id int64) (*AdminInstitutionRow, error) {
	query := `
		SELECT i.id, i.institution_name, i.institution_address, i.institution_code,
		       i.status, i.expire_time, i.subscription_plan_id, p.plan_name,
		       i.create_time, i.update_time
		FROM c_institution i
		LEFT JOIN c_subscription_plan p ON i.subscription_plan_id = p.id
		WHERE i.id = ?
	`
	row := m.db.QueryRow(query, id)
	inst, err := scanInstitution(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询机构失败: %w", err)
	}
	return inst, nil
}

// InsertInstitution 新增机构（对齐 Java insertInstitution）
//
// 流程：
//  1. 插入机构记录（institution_code 先占位为空）
//  2. 根据自增ID生成机构编码并更新
//
// 参数：
//   - name: 机构名称
//   - address: 机构地址
//   - subscriptionPlanID: 订阅套餐ID（0 默认为 1 标准套餐）
//   - expireTimeStr: 过期时间字符串（"yyyy-MM-dd HH:mm:ss"，空表示永久有效）
//
// 返回：新机构ID
func (m *AdminBusinessMapper) InsertInstitution(name, address string, subscriptionPlanID int64, expireTimeStr string) (int64, error) {
	// 套餐默认 1（标准套餐），对齐 Java dto.getSubscriptionPlanId() != null ? : 1L
	if subscriptionPlanID == 0 {
		subscriptionPlanID = 1
	}
	// 过期时间参数处理
	var expireArg interface{}
	if expireTimeStr != "" {
		expireArg = expireTimeStr
	}
	// 先插入占位码（空字符串，避免 NOT NULL 约束）
	query := `INSERT INTO c_institution (institution_name, institution_address, institution_code, status, subscription_plan_id, expire_time, create_time, update_time)
	          VALUES (?, ?, '', 1, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, name, address, subscriptionPlanID, expireArg)
	if err != nil {
		return 0, fmt.Errorf("新增机构失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取机构ID失败: %w", err)
	}

	// 根据自增ID生成机构编码（对齐 Java InstitutionCodeUtil.encodeToCode）
	// 简化实现：使用 "I" + 9位零填充ID，保证唯一性
	// 注意：与 Java Hashids 算法不完全一致，但满足"基于ID生成唯一码"的需求
	code := fmt.Sprintf("I%09d", id)
	if _, err := m.db.Exec(`UPDATE c_institution SET institution_code = ? WHERE id = ?`, code, id); err != nil {
		// 编码更新失败不阻断主流程（记录日志），机构已创建
		return id, fmt.Errorf("更新机构编码失败: %w", err)
	}
	return id, nil
}

// UpdateInstitution 更新机构（对齐 Java updateInstitution）
//
// 动态更新非空字段，支持清除过期时间（expireTimeStr 为 "NULL" 字符串时设为 NULL）
//
// 参数：
//   - id: 机构ID
//   - name: 机构名称（空不更新）
//   - address: 机构地址（空不更新）
//   - code: 机构编码（空不更新）
//   - status: 状态（0 不更新）
//   - subscriptionPlanID: 套餐ID（0 不更新）
//   - expireTimeStr: 过期时间（"" 不更新, "NULL" 设为永久有效, 其他值设为指定时间）
func (m *AdminBusinessMapper) UpdateInstitution(id int64, name, address, code string, status, subscriptionPlanID int64, expireTimeStr string) error {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if name != "" {
		setParts = append(setParts, "institution_name = ?")
		args = append(args, name)
	}
	if address != "" {
		setParts = append(setParts, "institution_address = ?")
		args = append(args, address)
	}
	if code != "" {
		setParts = append(setParts, "institution_code = ?")
		args = append(args, code)
	}
	if status != 0 {
		setParts = append(setParts, "status = ?")
		args = append(args, status)
	}
	if subscriptionPlanID != 0 {
		setParts = append(setParts, "subscription_plan_id = ?")
		args = append(args, subscriptionPlanID)
	}
	// 过期时间处理：对齐 Java 逻辑（空字符串=清除时间设为 NULL）
	if expireTimeStr != "" {
		if expireTimeStr == "NULL" {
			setParts = append(setParts, "expire_time = NULL")
		} else {
			setParts = append(setParts, "expire_time = ?")
			args = append(args, expireTimeStr)
		}
	}

	query := fmt.Sprintf("UPDATE c_institution SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新机构失败: %w", err)
	}
	return nil
}

// ============================================================
// 学生管理
// ============================================================

// AdminStudentRow 学生查询行（对齐 Java AdminStudentVO）
type AdminStudentRow struct {
	ID              int64                  `json:"id"`              // 学生ID
	Avatar          string                 `json:"avatar"`          // 头像URL
	StudentName     string                 `json:"studentName"`     // 学生姓名
	InstitutionID   int64                  `json:"institutionId"`   // 机构ID
	Sex             int64                  `json:"sex"`             // 性别（0=未知,1=男,2=女）
	BirthStr        string                 `json:"birthStr"`        // 出生日期字符串
	School          string                 `json:"school"`          // 学校
	Address         string                 `json:"address"`         // 地址
	CreateTimeStr   string                 `json:"createTimeStr"`   // 创建时间字符串
	UpdateTimeStr   string                 `json:"updateTimeStr"`   // 更新时间字符串
	PrimaryParent   *AdminParentInfoRow    `json:"primaryParent"`   // 主联系人家长（无则 nil）
	SecondaryParent *AdminParentInfoRow    `json:"secondaryParent"` // 次联系人家长（无则 nil）
}

// AdminParentInfoRow 家长信息行（对齐 Java ParentVO）
type AdminParentInfoRow struct {
	Username  string `json:"username"`  // 家长用户名
	ParentID  int64  `json:"parentId"`  // 家长ID
	StudentID int64  `json:"studentId"` // 学生ID
	Relation  string `json:"relation"`  // 关系（如"父亲"）
	Phone     string `json:"phone"`     // 手机号
	IsPrimary int64  `json:"isPrimary"` // 是否主联系人（1=是,0=否）
	IsBound   bool   `json:"isBound"`   // 是否已绑定微信
}

// ListStudents 学生分页列表（对齐 Java listStudents）
//
// 查询参数：
//   - institutionID: 机构ID（0 不过滤）
//   - keyword: 关键词（匹配学生姓名或学校，空不过滤）
//   - sex: 性别（-1 不过滤）
//   - offset/limit: 分页参数
//
// 返回：学生ID列表 + 总数（家长信息由 Service 层填充）
func (m *AdminBusinessMapper) ListStudents(institutionID int64, keyword string, sex int64, offset, limit int) ([]*AdminStudentRow, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if institutionID != 0 {
		where += " AND institution_id = ?"
		args = append(args, institutionID)
	}
	if keyword != "" {
		where += " AND (student_name LIKE ? OR school LIKE ?)"
		args = append(args, "%"+keyword+"%", "%"+keyword+"%")
	}
	if sex != -1 {
		where += " AND sex = ?"
		args = append(args, sex)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM c_student %s`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计学生数失败: %w", err)
	}

	// 分页查询
	query := fmt.Sprintf(`
		SELECT id, avatar, student_name, institution_id, sex, birth, school, address, create_time, update_time
		FROM c_student
		%s
		ORDER BY create_time DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询学生列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminStudentRow
	for rows.Next() {
		row := &AdminStudentRow{}
		var (
			avatar, name, school, address sql.NullString
			instID, sexVal                sql.NullInt64
			birth, createTime, updateTime sql.NullTime
		)
		if err := rows.Scan(&row.ID, &avatar, &name, &instID, &sexVal, &birth, &school, &address, &createTime, &updateTime); err != nil {
			return nil, 0, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		row.Avatar = avatar.String
		row.StudentName = name.String
		if instID.Valid {
			row.InstitutionID = instID.Int64
		}
		if sexVal.Valid {
			row.Sex = sexVal.Int64
		}
		row.BirthStr = formatDateSQL(birth)
		row.School = school.String
		row.Address = address.String
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, total, nil
}

// SelectStudentByID 按ID查学生
func (m *AdminBusinessMapper) SelectStudentByID(id int64) (*AdminStudentRow, error) {
	query := `SELECT id, avatar, student_name, institution_id, sex, birth, school, address, create_time, update_time FROM c_student WHERE id = ?`
	row := m.db.QueryRow(query, id)
	r := &AdminStudentRow{}
	var (
		avatar, name, school, address sql.NullString
		instID, sexVal                sql.NullInt64
		birth, createTime, updateTime sql.NullTime
	)
	if err := row.Scan(&r.ID, &avatar, &name, &instID, &sexVal, &birth, &school, &address, &createTime, &updateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询学生失败: %w", err)
	}
	r.Avatar = avatar.String
	r.StudentName = name.String
	if instID.Valid {
		r.InstitutionID = instID.Int64
	}
	if sexVal.Valid {
		r.Sex = sexVal.Int64
	}
	r.BirthStr = formatDateSQL(birth)
	r.School = school.String
	r.Address = address.String
	r.CreateTimeStr = formatTimeSQL(createTime)
	r.UpdateTimeStr = formatTimeSQL(updateTime)
	return r, nil
}

// InsertStudent 新增学生（对齐 Java insertStudent）
//
// 参数：
//   - studentName: 学生姓名
//   - institutionID: 机构ID
//   - sex: 性别（0=未知,1=男,2=女）
//   - birthStr: 出生日期（"yyyy-MM-dd"，空不设置）
//   - school: 学校
//   - address: 地址
//
// 返回：新学生ID
func (m *AdminBusinessMapper) InsertStudent(studentName string, institutionID, sex int64, birthStr, school, address string) (int64, error) {
	var birthArg interface{}
	if birthStr != "" {
		birthArg = birthStr
	}
	query := `INSERT INTO c_student (avatar, student_name, institution_id, sex, birth, school, address, create_time, update_time)
	          VALUES ('', ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, studentName, institutionID, sex, birthArg, school, address)
	if err != nil {
		return 0, fmt.Errorf("新增学生失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取学生ID失败: %w", err)
	}
	return id, nil
}

// UpdateStudent 更新学生（对齐 Java updateStudent，动态更新非空字段）
//
// 参数：
//   - id: 学生ID
//   - studentName: 学生姓名（空不更新）
//   - sex: 性别（-1 不更新）
//   - birthStr: 出生日期（"" 不更新, "NULL" 设为 NULL）
//   - school: 学校（空不更新）
//   - address: 地址（空不更新）
func (m *AdminBusinessMapper) UpdateStudent(id int64, studentName string, sex int64, birthStr, school, address string) error {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if studentName != "" {
		setParts = append(setParts, "student_name = ?")
		args = append(args, studentName)
	}
	if sex != -1 {
		setParts = append(setParts, "sex = ?")
		args = append(args, sex)
	}
	if birthStr != "" {
		if birthStr == "NULL" {
			setParts = append(setParts, "birth = NULL")
		} else {
			setParts = append(setParts, "birth = ?")
			args = append(args, birthStr)
		}
	}
	if school != "" {
		setParts = append(setParts, "school = ?")
		args = append(args, school)
	}
	if address != "" {
		setParts = append(setParts, "address = ?")
		args = append(args, address)
	}

	query := fmt.Sprintf("UPDATE c_student SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新学生失败: %w", err)
	}
	return nil
}

// SelectParentInfoByStudentID 按学生ID查询家长信息（主/次联系人）
//
// 对齐 Java listStudents 中注入家长信息的逻辑
// JOIN c_parent_student 和 c_parent 表
//
// 参数：
//   - studentID: 学生ID
//
// 返回：家长信息列表（一个学生可能有主/次两个家长）
func (m *AdminBusinessMapper) SelectParentInfoByStudentID(studentID int64) ([]*AdminParentInfoRow, error) {
	query := `
		SELECT ps.student_id, ps.parent_id, ps.is_primary, ps.relation,
		       p.username, p.phone, p.is_bound
		FROM c_parent_student ps
		INNER JOIN c_parent p ON ps.parent_id = p.id
		WHERE ps.student_id = ?
	`
	rows, err := m.db.Query(query, studentID)
	if err != nil {
		return nil, fmt.Errorf("查询学生家长信息失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminParentInfoRow
	for rows.Next() {
		row := &AdminParentInfoRow{}
		var (
			username, phone, relation sql.NullString
			parentID                  sql.NullInt64
			isPrimary                 sql.NullInt64
			isBound                   sql.NullBool
		)
		if err := rows.Scan(&row.StudentID, &parentID, &isPrimary, &relation, &username, &phone, &isBound); err != nil {
			return nil, fmt.Errorf("扫描家长信息失败: %w", err)
		}
		if parentID.Valid {
			row.ParentID = parentID.Int64
		}
		if isPrimary.Valid && isPrimary.Int64 == 1 {
			row.IsPrimary = 1
		}
		row.Relation = relation.String
		row.Username = username.String
		row.Phone = phone.String
		row.IsBound = isBound.Bool
		list = append(list, row)
	}
	return list, nil
}

// ============================================================
// 教师管理
// ============================================================

// AdminTeacherRow 教师查询行（对齐 Java AdminTeacherVO）
type AdminTeacherRow struct {
	TeacherID           int64  `json:"teacherId"`           // 教师ID
	InstitutionID       int64  `json:"institutionId"`       // 机构ID
	IsAvailable         bool   `json:"isAvailable"`         // 是否可用
	Username            string `json:"username"`            // 教师用户名
	Account             string `json:"account"`             // 登录账号（来自 c_user_auth）
	IsInstitutionAdmin  bool   `json:"isInstitutionAdmin"`  // 是否机构管理员
	LastLoginTimeStr    string `json:"lastLoginTimeStr"`    // 最后登录时间字符串
	CreateTimeStr       string `json:"createTimeStr"`       // 创建时间字符串（取自 c_user.create_time）
	UpdateTimeStr       string `json:"updateTimeStr"`       // 更新时间字符串
}

// ListTeachers 教师分页列表（对齐 Java listTeachers）
//
// JOIN c_user_auth 获取登录账号和最后登录时间
// JOIN c_user 获取创建时间（教师表无 create_time）
//
// 查询参数：
//   - institutionID: 机构ID（0 不过滤）
//   - keyword: 关键词（匹配教师用户名，空不过滤）—— 对齐 Java 的 isAvailable 过滤
//   - isAvailable: 是否可用（-1 不过滤,0=不可用,1=可用）
//   - offset/limit: 分页参数
func (m *AdminBusinessMapper) ListTeachers(institutionID int64, keyword string, isAvailable int64, offset, limit int) ([]*AdminTeacherRow, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if institutionID != 0 {
		where += " AND t.institution_id = ?"
		args = append(args, institutionID)
	}
	if isAvailable != -1 {
		where += " AND t.is_available = ?"
		args = append(args, isAvailable == 1)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM c_teacher t %s`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计教师数失败: %w", err)
	}

	// 分页查询（LEFT JOIN c_user_auth 获取账号信息）
	query := fmt.Sprintf(`
		SELECT t.id, t.institution_id, t.is_available, t.username, t.is_institution_admin,
		       ua.account, ua.last_login_time, u.create_time, t.id
		FROM c_teacher t
		LEFT JOIN c_user u ON t.user_id = u.id
		LEFT JOIN c_user_auth ua ON ua.user_id = u.id AND ua.role_id = 4
		%s
		ORDER BY t.id DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询教师列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminTeacherRow
	for rows.Next() {
		row := &AdminTeacherRow{}
		var (
			instID                              sql.NullInt64
			isAvailable, isInstitutionAdmin     sql.NullBool
			username, account                   sql.NullString
			lastLoginTime, createTime           sql.NullTime
		)
		if err := rows.Scan(&row.TeacherID, &instID, &isAvailable, &username, &isInstitutionAdmin, &account, &lastLoginTime, &createTime, &row.TeacherID); err != nil {
			return nil, 0, fmt.Errorf("扫描教师记录失败: %w", err)
		}
		if instID.Valid {
			row.InstitutionID = instID.Int64
		}
		row.IsAvailable = isAvailable.Bool
		row.Username = username.String
		row.IsInstitutionAdmin = isInstitutionAdmin.Bool
		row.Account = account.String
		row.LastLoginTimeStr = formatTimeSQL(lastLoginTime)
		row.CreateTimeStr = formatTimeSQL(createTime)
		list = append(list, row)
	}
	return list, total, nil
}

// SelectTeacherByID 按ID查教师
func (m *AdminBusinessMapper) SelectTeacherByID(teacherID int64) (*AdminTeacherRow, error) {
	query := `
		SELECT t.id, t.institution_id, t.is_available, t.username, t.is_institution_admin,
		       ua.account, ua.last_login_time, u.create_time, t.id
		FROM c_teacher t
		LEFT JOIN c_user u ON t.user_id = u.id
		LEFT JOIN c_user_auth ua ON ua.user_id = u.id AND ua.role_id = 4
		WHERE t.id = ?
	`
	row := m.db.QueryRow(query, teacherID)
	r := &AdminTeacherRow{}
	var (
		instID                              sql.NullInt64
		isAvailable, isInstitutionAdmin     sql.NullBool
		username, account                   sql.NullString
		lastLoginTime, createTime           sql.NullTime
	)
	if err := row.Scan(&r.TeacherID, &instID, &isAvailable, &username, &isInstitutionAdmin, &account, &lastLoginTime, &createTime, &r.TeacherID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询教师失败: %w", err)
	}
	if instID.Valid {
		r.InstitutionID = instID.Int64
	}
	r.IsAvailable = isAvailable.Bool
	r.Username = username.String
	r.IsInstitutionAdmin = isInstitutionAdmin.Bool
	r.Account = account.String
	r.LastLoginTimeStr = formatTimeSQL(lastLoginTime)
	r.CreateTimeStr = formatTimeSQL(createTime)
	return r, nil
}

// InsertTeacher 新增教师（对齐 Java insertTeacher）
//
// 流程：
//  1. 创建 c_user 记录（teacher.user_id 外键）
//  2. 创建 c_teacher 记录
//
// 注意：账号密码创建由 Service 层调用 UserAuth 相关方法处理
//
// 参数：
//   - username: 教师用户名
//   - institutionID: 机构ID
//   - isAvailable: 是否可用
//
// 返回：新教师ID, 新用户ID, 错误
func (m *AdminBusinessMapper) InsertTeacher(username string, institutionID int64, isAvailable bool) (teacherID, userID int64, err error) {
	// 1. 创建 user 记录
	userResult, err := m.db.Exec(`INSERT INTO c_user (institution_id, create_time, update_time) VALUES (?, NOW(), NOW())`, institutionID)
	if err != nil {
		return 0, 0, fmt.Errorf("新增用户记录失败: %w", err)
	}
	userID, err = userResult.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("获取用户ID失败: %w", err)
	}

	// 2. 创建 teacher 记录
	teacherResult, err := m.db.Exec(`INSERT INTO c_teacher (user_id, is_available, username, institution_id, is_institution_admin) VALUES (?, ?, ?, ?, 0)`, userID, isAvailable, username, institutionID)
	if err != nil {
		return 0, userID, fmt.Errorf("新增教师失败: %w", err)
	}
	teacherID, err = teacherResult.LastInsertId()
	if err != nil {
		return 0, userID, fmt.Errorf("获取教师ID失败: %w", err)
	}
	return teacherID, userID, nil
}

// UpdateTeacher 更新教师（对齐 Java updateTeacher）
//
// 参数：
//   - teacherID: 教师ID
//   - username: 用户名（空不更新）
//   - isAvailable: 是否可用（-1 不更新,0=不可用,1=可用）
func (m *AdminBusinessMapper) UpdateTeacher(teacherID int64, username string, isAvailable int64) error {
	setParts := []string{}
	args := []interface{}{}
	if username != "" {
		setParts = append(setParts, "username = ?")
		args = append(args, username)
	}
	if isAvailable != -1 {
		setParts = append(setParts, "is_available = ?")
		args = append(args, isAvailable == 1)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "id = id") // 保证 SET 子句非空
	query := fmt.Sprintf("UPDATE c_teacher SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, teacherID)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新教师失败: %w", err)
	}
	return nil
}

// CountClassTeacherByTeacherID 统计教师关联的班级数（用于校验教师是否可删除）
func (m *AdminBusinessMapper) CountClassTeacherByTeacherID(teacherID int64) (int64, error) {
	var count int64
	if err := m.db.QueryRow(`SELECT COUNT(1) FROM c_class_teacher WHERE teacher_id = ?`, teacherID).Scan(&count); err != nil {
		return 0, fmt.Errorf("统计教师班级数失败: %w", err)
	}
	return count, nil
}

// ============================================================
// 课程管理
// ============================================================

// AdminCourseRow 课程查询行（对齐 Java AdminCourseVO）
type AdminCourseRow struct {
	ID            int64  `json:"id"`            // 课程ID
	CourseName    string `json:"courseName"`    // 课程名称
	CourseType    int64  `json:"courseType"`    // 课程类型（1=按次, 2=按天）
	InstitutionID int64  `json:"institutionId"` // 机构ID
	IsAvailable   bool   `json:"isAvailable"`   // 是否可用
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
}

// ListCourses 课程分页列表（对齐 Java listCourses）
func (m *AdminBusinessMapper) ListCourses(institutionID int64, keyword string, courseType, isAvailable int64, offset, limit int) ([]*AdminCourseRow, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if institutionID != 0 {
		where += " AND institution_id = ?"
		args = append(args, institutionID)
	}
	if keyword != "" {
		where += " AND course_name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	if courseType != 0 {
		where += " AND course_type = ?"
		args = append(args, courseType)
	}
	if isAvailable != -1 {
		where += " AND is_available = ?"
		args = append(args, isAvailable == 1)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM c_course %s`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计课程数失败: %w", err)
	}

	// 分页查询
	query := fmt.Sprintf(`
		SELECT id, course_name, course_type, institution_id, is_available, create_time, update_time
		FROM c_course
		%s
		ORDER BY create_time DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课程列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminCourseRow
	for rows.Next() {
		row := &AdminCourseRow{}
		var (
			name                          sql.NullString
			cType, instID                 sql.NullInt64
			isAvail                       sql.NullBool
			createTime, updateTime        sql.NullTime
		)
		if err := rows.Scan(&row.ID, &name, &cType, &instID, &isAvail, &createTime, &updateTime); err != nil {
			return nil, 0, fmt.Errorf("扫描课程记录失败: %w", err)
		}
		row.CourseName = name.String
		if cType.Valid {
			row.CourseType = cType.Int64
		}
		if instID.Valid {
			row.InstitutionID = instID.Int64
		}
		row.IsAvailable = isAvail.Bool
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, total, nil
}

// SelectCourseByID 按ID查课程
func (m *AdminBusinessMapper) SelectCourseByID(id int64) (*AdminCourseRow, error) {
	query := `SELECT id, course_name, course_type, institution_id, is_available, create_time, update_time FROM c_course WHERE id = ?`
	row := m.db.QueryRow(query, id)
	r := &AdminCourseRow{}
	var (
		name                          sql.NullString
		cType, instID                 sql.NullInt64
		isAvail                       sql.NullBool
		createTime, updateTime        sql.NullTime
	)
	if err := row.Scan(&r.ID, &name, &cType, &instID, &isAvail, &createTime, &updateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课程失败: %w", err)
	}
	r.CourseName = name.String
	if cType.Valid {
		r.CourseType = cType.Int64
	}
	if instID.Valid {
		r.InstitutionID = instID.Int64
	}
	r.IsAvailable = isAvail.Bool
	r.CreateTimeStr = formatTimeSQL(createTime)
	r.UpdateTimeStr = formatTimeSQL(updateTime)
	return r, nil
}

// InsertCourse 新增课程
func (m *AdminBusinessMapper) InsertCourse(courseName string, courseType, institutionID int64, isAvailable bool) (int64, error) {
	query := `INSERT INTO c_course (course_name, course_type, institution_id, is_available, create_time, update_time)
	          VALUES (?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, courseName, courseType, institutionID, isAvailable)
	if err != nil {
		return 0, fmt.Errorf("新增课程失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取课程ID失败: %w", err)
	}
	return id, nil
}

// UpdateCourse 更新课程（动态更新非空字段）
func (m *AdminBusinessMapper) UpdateCourse(id int64, courseName string, courseType, isAvailable int64) error {
	setParts := []string{}
	args := []interface{}{}
	if courseName != "" {
		setParts = append(setParts, "course_name = ?")
		args = append(args, courseName)
	}
	if courseType != 0 {
		setParts = append(setParts, "course_type = ?")
		args = append(args, courseType)
	}
	if isAvailable != -1 {
		setParts = append(setParts, "is_available = ?")
		args = append(args, isAvailable == 1)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "update_time = NOW()")
	query := fmt.Sprintf("UPDATE c_course SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新课程失败: %w", err)
	}
	return nil
}

// ============================================================
// 班级管理
// ============================================================

// AdminClassRow 班级查询行（对齐 Java AdminClassVO + ClassResponse）
type AdminClassRow struct {
	ID              int64    `json:"id"`              // 班级ID
	CourseID        int64    `json:"courseId"`        // 课程ID
	ClassName       string   `json:"className"`       // 班级名称
	Status          int64    `json:"status"`          // 班级状态
	StudentCount    int64    `json:"studentCount"`    // 班级学生人数
	StudentMaxCount int64    `json:"studentMaxCount"` // 班级最大人数
	TeacherIDs      []int64  `json:"teacherIds"`      // 教师ID列表
	TeacherNames    []string `json:"teacherNames"`    // 教师用户名列表
	CreateTimeStr   string   `json:"createTimeStr"`   // 创建时间字符串
	UpdateTimeStr   string   `json:"updateTimeStr"`   // 更新时间字符串
}

// ListClasses 班级分页列表（对齐 Java listClasses）
//
// 查询参数：
//   - courseID: 课程ID（0 不过滤）
//   - keyword: 班级名称关键词（空不过滤）
//   - status: 班级状态（-1 不过滤）
//   - institutionID: 机构ID（0 不过滤，通过 JOIN c_course 过滤）
//   - offset/limit: 分页参数
func (m *AdminBusinessMapper) ListClasses(courseID int64, keyword string, status, institutionID int64, offset, limit int) ([]*AdminClassRow, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if courseID != 0 {
		where += " AND c.course_id = ?"
		args = append(args, courseID)
	}
	if keyword != "" {
		where += " AND c.class_name LIKE ?"
		args = append(args, "%"+keyword+"%")
	}
	if status != -1 {
		where += " AND c.status = ?"
		args = append(args, status)
	}
	if institutionID != 0 {
		where += " AND co.institution_id = ?"
		args = append(args, institutionID)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`
		SELECT COUNT(1) FROM c_class c
		LEFT JOIN c_course co ON c.course_id = co.id
		%s
	`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计班级数失败: %w", err)
	}

	// 分页查询（不含教师信息，教师信息由 Service 层填充）
	query := fmt.Sprintf(`
		SELECT c.id, c.course_id, c.class_name, c.status, c.student_count, c.student_max_count,
		       c.create_time, c.update_time
		FROM c_class c
		LEFT JOIN c_course co ON c.course_id = co.id
		%s
		ORDER BY c.create_time DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询班级列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminClassRow
	for rows.Next() {
		row := &AdminClassRow{
			TeacherIDs:   []int64{},
			TeacherNames: []string{},
		}
		var (
			className                          sql.NullString
			courseID, status, studentCount, studentMax sql.NullInt64
			createTime, updateTime             sql.NullTime
		)
		if err := rows.Scan(&row.ID, &courseID, &className, &status, &studentCount, &studentMax, &createTime, &updateTime); err != nil {
			return nil, 0, fmt.Errorf("扫描班级记录失败: %w", err)
		}
		if courseID.Valid {
			row.CourseID = courseID.Int64
		}
		row.ClassName = className.String
		if status.Valid {
			row.Status = status.Int64
		}
		if studentCount.Valid {
			row.StudentCount = studentCount.Int64
		}
		if studentMax.Valid {
			row.StudentMaxCount = studentMax.Int64
		}
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, total, nil
}

// SelectClassTeachers 按班级ID查询教师列表
//
// JOIN c_teacher 获取教师用户名
func (m *AdminBusinessMapper) SelectClassTeachers(classID int64) ([]int64, []string, error) {
	query := `
		SELECT t.id, t.username
		FROM c_class_teacher ct
		INNER JOIN c_teacher t ON ct.teacher_id = t.id
		WHERE ct.class_id = ?
	`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, nil, fmt.Errorf("查询班级教师失败: %w", err)
	}
	defer rows.Close()

	var ids []int64
	var names []string
	for rows.Next() {
		var id sql.NullInt64
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return nil, nil, fmt.Errorf("扫描班级教师失败: %w", err)
		}
		if id.Valid {
			ids = append(ids, id.Int64)
		}
		names = append(names, name.String)
	}
	if ids == nil {
		ids = []int64{}
	}
	if names == nil {
		names = []string{}
	}
	return ids, names, nil
}

// SelectClassByID 按ID查班级
func (m *AdminBusinessMapper) SelectClassByID(classID int64) (*AdminClassRow, error) {
	query := `SELECT id, course_id, class_name, status, student_count, student_max_count, create_time, update_time FROM c_class WHERE id = ?`
	row := m.db.QueryRow(query, classID)
	r := &AdminClassRow{
		TeacherIDs:   []int64{},
		TeacherNames: []string{},
	}
	var (
		className                          sql.NullString
		courseID, status, studentCount, studentMax sql.NullInt64
		createTime, updateTime             sql.NullTime
	)
	if err := row.Scan(&r.ID, &courseID, &className, &status, &studentCount, &studentMax, &createTime, &updateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询班级失败: %w", err)
	}
	if courseID.Valid {
		r.CourseID = courseID.Int64
	}
	r.ClassName = className.String
	if status.Valid {
		r.Status = status.Int64
	}
	if studentCount.Valid {
		r.StudentCount = studentCount.Int64
	}
	if studentMax.Valid {
		r.StudentMaxCount = studentMax.Int64
	}
	r.CreateTimeStr = formatTimeSQL(createTime)
	r.UpdateTimeStr = formatTimeSQL(updateTime)
	return r, nil
}

// InsertClass 新增班级
//
// 返回：班级ID
func (m *AdminBusinessMapper) InsertClass(className string, courseID, studentMaxCount, status int64) (int64, error) {
	query := `INSERT INTO c_class (course_id, class_name, student_max_count, status, student_count, create_time, update_time)
	          VALUES (?, ?, ?, ?, 0, NOW(), NOW())`
	result, err := m.db.Exec(query, courseID, className, studentMaxCount, status)
	if err != nil {
		return 0, fmt.Errorf("新增班级失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取班级ID失败: %w", err)
	}
	return id, nil
}

// UpdateClass 更新班级（动态更新非空字段）
func (m *AdminBusinessMapper) UpdateClass(classID int64, className string, studentMaxCount, status int64) error {
	setParts := []string{}
	args := []interface{}{}
	if className != "" {
		setParts = append(setParts, "class_name = ?")
		args = append(args, className)
	}
	if studentMaxCount != 0 {
		setParts = append(setParts, "student_max_count = ?")
		args = append(args, studentMaxCount)
	}
	if status != -1 {
		setParts = append(setParts, "status = ?")
		args = append(args, status)
	}
	if len(setParts) == 0 {
		return nil
	}
	setParts = append(setParts, "update_time = NOW()")
	query := fmt.Sprintf("UPDATE c_class SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, classID)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新班级失败: %w", err)
	}
	return nil
}

// InsertClassStudent 班级添加学生（单个）
func (m *AdminBusinessMapper) InsertClassStudent(classID, studentID int64) error {
	_, err := m.db.Exec(`INSERT INTO c_class_student (class_id, student_id, create_time) VALUES (?, ?, NOW())`, classID, studentID)
	if err != nil {
		return fmt.Errorf("添加班级学生失败: %w", err)
	}
	return nil
}

// DeleteClassStudent 班级移除学生（单个）
func (m *AdminBusinessMapper) DeleteClassStudent(classID, studentID int64) error {
	_, err := m.db.Exec(`DELETE FROM c_class_student WHERE class_id = ? AND student_id = ?`, classID, studentID)
	if err != nil {
		return fmt.Errorf("移除班级学生失败: %w", err)
	}
	return nil
}

// UpdateClassStudentCount 更新班级学生人数（统计 c_class_student 表）
func (m *AdminBusinessMapper) UpdateClassStudentCount(classID int64) error {
	_, err := m.db.Exec(`UPDATE c_class SET student_count = (SELECT COUNT(*) FROM c_class_student WHERE class_id = ?), update_time = NOW() WHERE id = ?`, classID, classID)
	if err != nil {
		return fmt.Errorf("更新班级学生人数失败: %w", err)
	}
	return nil
}

// SelectStudentsByClassID 按班级ID查询学生列表（用于 class/get_by_id）
func (m *AdminBusinessMapper) SelectStudentsByClassID(classID int64) ([]*AdminStudentRow, error) {
	query := `
		SELECT s.id, s.avatar, s.student_name, s.institution_id, s.sex, s.birth, s.school, s.address, s.create_time, s.update_time
		FROM c_class_student cs
		INNER JOIN c_student s ON cs.student_id = s.id
		WHERE cs.class_id = ?
	`
	rows, err := m.db.Query(query, classID)
	if err != nil {
		return nil, fmt.Errorf("查询班级学生失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminStudentRow
	for rows.Next() {
		row := &AdminStudentRow{}
		var (
			avatar, name, school, address sql.NullString
			instID, sexVal                sql.NullInt64
			birth, createTime, updateTime sql.NullTime
		)
		if err := rows.Scan(&row.ID, &avatar, &name, &instID, &sexVal, &birth, &school, &address, &createTime, &updateTime); err != nil {
			return nil, fmt.Errorf("扫描学生记录失败: %w", err)
		}
		row.Avatar = avatar.String
		row.StudentName = name.String
		if instID.Valid {
			row.InstitutionID = instID.Int64
		}
		if sexVal.Valid {
			row.Sex = sexVal.Int64
		}
		row.BirthStr = formatDateSQL(birth)
		row.School = school.String
		row.Address = address.String
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, nil
}

// ============================================================
// 课表管理
// ============================================================

// AdminClassScheduleRow 课表查询行（对齐 Java AdminClassScheduleVO）
type AdminClassScheduleRow struct {
	ID            int64  `json:"id"`            // 课表ID
	ClassID       int64  `json:"classId"`       // 班级ID
	StartDateStr  string `json:"startDateStr"`  // 开始日期字符串
	EndDateStr    string `json:"endDateStr"`    // 结束日期字符串
	DayOfWeek     int64  `json:"dayOfWeek"`     // 星期几（1-7）
	StartTimeStr  string `json:"startTimeStr"`  // 开始时间字符串
	EndTimeStr    string `json:"endTimeStr"`    // 结束时间字符串
	Remark        string `json:"remark"`        // 备注
	CreateTimeStr string `json:"createTime"`    // 创建时间字符串
	UpdateTimeStr string `json:"updateTime"`    // 更新时间字符串
}

// ListClassSchedules 课表列表（对齐 Java listClassSchedules）
//
// 注意：本接口不分页，返回所有匹配记录
//
// 查询参数：
//   - classID: 班级ID（0 不过滤）
//   - dayOfWeek: 星期几（0 不过滤）
//   - institutionID: 机构ID（0 不过滤，通过班级→课程→机构过滤）
func (m *AdminBusinessMapper) ListClassSchedules(classID, dayOfWeek, institutionID int64) ([]*AdminClassScheduleRow, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if classID != 0 {
		where += " AND cs.class_id = ?"
		args = append(args, classID)
	}
	if dayOfWeek != 0 {
		where += " AND cs.day_of_week = ?"
		args = append(args, dayOfWeek)
	}
	if institutionID != 0 {
		// 通过班级→课程→机构过滤
		where += " AND EXISTS (SELECT 1 FROM c_class c JOIN c_course co ON c.course_id = co.id WHERE c.id = cs.class_id AND co.institution_id = ?)"
		args = append(args, institutionID)
	}

	query := fmt.Sprintf(`
		SELECT cs.id, cs.class_id, cs.start_date, cs.end_date, cs.day_of_week,
		       cs.start_time, cs.end_time, cs.remark, cs.create_time, cs.update_time
		FROM c_class_schedule cs
		%s
		ORDER BY cs.day_of_week ASC, cs.start_time ASC
	`, where)
	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询课表列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminClassScheduleRow
	for rows.Next() {
		row := &AdminClassScheduleRow{}
		var (
			classID                                  sql.NullInt64
			dayOfWeek                                sql.NullInt64
			startDate, endDate, startTime, endTime   sql.NullTime
			remark                                   sql.NullString
			createTime, updateTime                   sql.NullTime
		)
		if err := rows.Scan(&row.ID, &classID, &startDate, &endDate, &dayOfWeek, &startTime, &endTime, &remark, &createTime, &updateTime); err != nil {
			return nil, fmt.Errorf("扫描课表记录失败: %w", err)
		}
		if classID.Valid {
			row.ClassID = classID.Int64
		}
		row.StartDateStr = formatDateSQL(startDate)
		row.EndDateStr = formatDateSQL(endDate)
		if dayOfWeek.Valid {
			row.DayOfWeek = dayOfWeek.Int64
		}
		// 开始/结束时间只取 HH:mm:ss 部分
		row.StartTimeStr = formatTimePart(startTime)
		row.EndTimeStr = formatTimePart(endTime)
		row.Remark = remark.String
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, nil
}

// SelectClassScheduleByID 按ID查课表
func (m *AdminBusinessMapper) SelectClassScheduleByID(id int64) (*AdminClassScheduleRow, error) {
	query := `SELECT id, class_id, start_date, end_date, day_of_week, start_time, end_time, remark, create_time, update_time FROM c_class_schedule WHERE id = ?`
	row := m.db.QueryRow(query, id)
	r := &AdminClassScheduleRow{}
	var (
		classID                                  sql.NullInt64
		dayOfWeek                                sql.NullInt64
		startDate, endDate, startTime, endTime   sql.NullTime
		remark                                   sql.NullString
		createTime, updateTime                   sql.NullTime
	)
	if err := row.Scan(&r.ID, &classID, &startDate, &endDate, &dayOfWeek, &startTime, &endTime, &remark, &createTime, &updateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课表失败: %w", err)
	}
	if classID.Valid {
		r.ClassID = classID.Int64
	}
	r.StartDateStr = formatDateSQL(startDate)
	r.EndDateStr = formatDateSQL(endDate)
	if dayOfWeek.Valid {
		r.DayOfWeek = dayOfWeek.Int64
	}
	r.StartTimeStr = formatTimePart(startTime)
	r.EndTimeStr = formatTimePart(endTime)
	r.Remark = remark.String
	r.CreateTimeStr = formatTimeSQL(createTime)
	r.UpdateTimeStr = formatTimeSQL(updateTime)
	return r, nil
}

// UpdateClassSchedule 更新课表（对齐 Java updateClassSchedule）
//
// 注意：Java 仅更新 day_of_week，本实现支持全字段动态更新
//
// 参数：
//   - id: 课表ID
//   - startDateStr: 开始日期（"" 不更新）
//   - endDateStr: 结束日期（"" 不更新）
//   - dayOfWeek: 星期几（0 不更新）
//   - startTimeStr: 开始时间（"" 不更新）
//   - endTimeStr: 结束时间（"" 不更新）
//   - remark: 备注（"" 不更新, "NULL" 设为 NULL）
func (m *AdminBusinessMapper) UpdateClassSchedule(id int64, startDateStr, endDateStr string, dayOfWeek int64, startTimeStr, endTimeStr, remark string) error {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if startDateStr != "" {
		setParts = append(setParts, "start_date = ?")
		args = append(args, startDateStr)
	}
	if endDateStr != "" {
		setParts = append(setParts, "end_date = ?")
		args = append(args, endDateStr)
	}
	if dayOfWeek != 0 {
		setParts = append(setParts, "day_of_week = ?")
		args = append(args, dayOfWeek)
	}
	if startTimeStr != "" {
		setParts = append(setParts, "start_time = ?")
		args = append(args, startTimeStr)
	}
	if endTimeStr != "" {
		setParts = append(setParts, "end_time = ?")
		args = append(args, endTimeStr)
	}
	if remark != "" {
		if remark == "NULL" {
			setParts = append(setParts, "remark = NULL")
		} else {
			setParts = append(setParts, "remark = ?")
			args = append(args, remark)
		}
	}

	query := fmt.Sprintf("UPDATE c_class_schedule SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新课表失败: %w", err)
	}
	return nil
}

// ============================================================
// 课时记录管理（c_course_record）
// ============================================================

// AdminCourseRecordRow 课时记录查询行（对齐 Java AdminCourseRecordVO）
type AdminCourseRecordRow struct {
	ID                int64  `json:"id"`                // 课卡记录ID
	StudentID         int64  `json:"studentId"`         // 学生ID
	CourseID          int64  `json:"courseId"`          // 课程ID
	CourseTotalTime   int64  `json:"courseTotalTime"`   // 课时总数
	CourseRestTime    int64  `json:"courseRestTime"`    // 剩余课时
	CourseLastTimeStr string `json:"courseLastTimeStr"` // 上次上课时间字符串
	ExpireTimeStr     string `json:"expireTimeStr"`     // 过期时间字符串
	CourseStatus      int64  `json:"courseStatus"`      // 课程状态
	CourseOwnerUserID int64  `json:"courseOwnerUserId"` // 课程归属人ID
	CourseRemark      string `json:"courseRemark"`      // 课程备注
	IsDelete          bool   `json:"isDelete"`          // 是否已删除
	CreateTimeStr     string `json:"createTimeStr"`     // 创建时间字符串
	UpdateTimeStr     string `json:"updateTimeStr"`     // 更新时间字符串
}

// ListCourseRecords 课时记录分页列表（对齐 Java listCourseRecords）
//
// 查询参数：
//   - studentID: 学生ID（0 不过滤）
//   - courseID: 课程ID（0 不过滤）
//   - courseStatus: 课程状态（0 不过滤）
//   - institutionID: 机构ID（0 不过滤，通过课程→机构过滤）
//   - offset/limit: 分页参数
func (m *AdminBusinessMapper) ListCourseRecords(studentID, courseID, courseStatus, institutionID int64, offset, limit int) ([]*AdminCourseRecordRow, int64, error) {
	where := "WHERE cr.is_delete = 0"
	args := []interface{}{}
	if studentID != 0 {
		where += " AND cr.student_id = ?"
		args = append(args, studentID)
	}
	if courseID != 0 {
		where += " AND cr.course_id = ?"
		args = append(args, courseID)
	}
	if courseStatus != 0 {
		where += " AND cr.course_status = ?"
		args = append(args, courseStatus)
	}
	if institutionID != 0 {
		where += " AND EXISTS (SELECT 1 FROM c_course co WHERE co.id = cr.course_id AND co.institution_id = ?)"
		args = append(args, institutionID)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM c_course_record cr %s`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计课时记录数失败: %w", err)
	}

	// 分页查询
	query := fmt.Sprintf(`
		SELECT cr.id, cr.student_id, cr.course_id, cr.course_total_time, cr.course_rest_time,
		       cr.course_last_time, cr.expire_time, cr.course_status, cr.course_owner_user_id,
		       cr.course_remark, cr.is_delete, cr.create_time, cr.update_time
		FROM c_course_record cr
		%s
		ORDER BY cr.create_time DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询课时记录列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminCourseRecordRow
	for rows.Next() {
		row := &AdminCourseRecordRow{}
		var (
			studentID, courseID, totalTime, restTime, status, ownerID sql.NullInt64
			lastTime, expireTime, createTime, updateTime             sql.NullTime
			remark                                                   sql.NullString
			isDelete                                                 sql.NullBool
		)
		if err := rows.Scan(&row.ID, &studentID, &courseID, &totalTime, &restTime, &lastTime, &expireTime, &status, &ownerID, &remark, &isDelete, &createTime, &updateTime); err != nil {
			return nil, 0, fmt.Errorf("扫描课时记录失败: %w", err)
		}
		if studentID.Valid {
			row.StudentID = studentID.Int64
		}
		if courseID.Valid {
			row.CourseID = courseID.Int64
		}
		if totalTime.Valid {
			row.CourseTotalTime = totalTime.Int64
		}
		if restTime.Valid {
			row.CourseRestTime = restTime.Int64
		}
		row.CourseLastTimeStr = formatTimeSQL(lastTime)
		row.ExpireTimeStr = formatTimeSQL(expireTime)
		if status.Valid {
			row.CourseStatus = status.Int64
		}
		if ownerID.Valid {
			row.CourseOwnerUserID = ownerID.Int64
		}
		row.CourseRemark = remark.String
		row.IsDelete = isDelete.Bool
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, total, nil
}

// SelectCourseRecordByID 按ID查课时记录
func (m *AdminBusinessMapper) SelectCourseRecordByID(id int64) (*AdminCourseRecordRow, error) {
	query := `SELECT id, student_id, course_id, course_total_time, course_rest_time, course_last_time, expire_time, course_status, course_owner_user_id, course_remark, is_delete, create_time, update_time FROM c_course_record WHERE id = ?`
	row := m.db.QueryRow(query, id)
	r := &AdminCourseRecordRow{}
	var (
		studentID, courseID, totalTime, restTime, status, ownerID sql.NullInt64
		lastTime, expireTime, createTime, updateTime             sql.NullTime
		remark                                                   sql.NullString
		isDelete                                                 sql.NullBool
	)
	if err := row.Scan(&r.ID, &studentID, &courseID, &totalTime, &restTime, &lastTime, &expireTime, &status, &ownerID, &remark, &isDelete, &createTime, &updateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询课时记录失败: %w", err)
	}
	if studentID.Valid {
		r.StudentID = studentID.Int64
	}
	if courseID.Valid {
		r.CourseID = courseID.Int64
	}
	if totalTime.Valid {
		r.CourseTotalTime = totalTime.Int64
	}
	if restTime.Valid {
		r.CourseRestTime = restTime.Int64
	}
	r.CourseLastTimeStr = formatTimeSQL(lastTime)
	r.ExpireTimeStr = formatTimeSQL(expireTime)
	if status.Valid {
		r.CourseStatus = status.Int64
	}
	if ownerID.Valid {
		r.CourseOwnerUserID = ownerID.Int64
	}
	r.CourseRemark = remark.String
	r.IsDelete = isDelete.Bool
	r.CreateTimeStr = formatTimeSQL(createTime)
	r.UpdateTimeStr = formatTimeSQL(updateTime)
	return r, nil
}

// InsertCourseRecord 新增课时记录（对齐 Java insertCourseRecord）
//
// 参数：
//   - studentID: 学生ID
//   - courseID: 课程ID
//   - totalTime: 课时总数
//   - restTime: 剩余课时（0 默认为 totalTime）
//   - expireTimeStr: 过期时间（"" 永久有效）
//   - status: 课程状态（0 默认为 1）
//   - remark: 备注
//
// 返回：新记录ID
func (m *AdminBusinessMapper) InsertCourseRecord(studentID, courseID, totalTime, restTime, status int64, expireTimeStr, remark string) (int64, error) {
	// 默认值处理（对齐 Java dto.getCourseRestTime() != null ? : totalTime）
	if restTime == 0 {
		restTime = totalTime
	}
	if status == 0 {
		status = 1
	}
	var expireArg interface{}
	if expireTimeStr != "" {
		expireArg = expireTimeStr
	}
	query := `INSERT INTO c_course_record (student_id, course_id, course_total_time, course_rest_time, course_status, expire_time, course_remark, is_delete, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, ?, ?, 0, NOW(), NOW())`
	result, err := m.db.Exec(query, studentID, courseID, totalTime, restTime, status, expireArg, remark)
	if err != nil {
		return 0, fmt.Errorf("新增课时记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取课时记录ID失败: %w", err)
	}
	return id, nil
}

// UpdateCourseRecord 更新课时记录（对齐 Java updateCourseRecord，动态更新非空字段）
//
// 参数：
//   - id: 记录ID
//   - restTime: 剩余课时（-1 不更新）
//   - totalTime: 总课时（-1 不更新）
//   - status: 状态（-1 不更新）
//   - remark: 备注（"" 不更新, "NULL" 设为 NULL）
func (m *AdminBusinessMapper) UpdateCourseRecord(id int64, restTime, totalTime, status int64, remark string) error {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if restTime != -1 {
		setParts = append(setParts, "course_rest_time = ?")
		args = append(args, restTime)
	}
	if totalTime != -1 {
		setParts = append(setParts, "course_total_time = ?")
		args = append(args, totalTime)
	}
	if status != -1 {
		setParts = append(setParts, "course_status = ?")
		args = append(args, status)
	}
	if remark != "" {
		if remark == "NULL" {
			setParts = append(setParts, "course_remark = NULL")
		} else {
			setParts = append(setParts, "course_remark = ?")
			args = append(args, remark)
		}
	}

	query := fmt.Sprintf("UPDATE c_course_record SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新课时记录失败: %w", err)
	}
	return nil
}

// ============================================================
// 上课记录管理（c_record）
// ============================================================

// AdminRecordRow 上课记录查询行（对齐 Java AdminRecordVO）
type AdminRecordRow struct {
	ID              int64  `json:"id"`              // 记录ID
	CourseRecordID  int64  `json:"courseRecordId"`  // 课卡记录ID
	RecordTimeStr   string `json:"recordTimeStr"`   // 记录时间字符串
	RecordRemark    string `json:"recordRemark"`    // 备注
	RecordType      int64  `json:"recordType"`      // 记录类型（1=增加, 2=减少）
	RecordChange    int64  `json:"recordChange"`    // 课时变更数量
	OperateTeacherID int64 `json:"operateTeacherId"` // 操作人ID（c_teacher.id）
	CreateTimeStr   string `json:"createTimeStr"`   // 创建时间字符串
	UpdateTimeStr   string `json:"updateTimeStr"`   // 更新时间字符串
}

// ListRecords 上课记录分页列表（对齐 Java listRecords）
//
// 查询参数：
//   - courseRecordID: 课卡记录ID（0 不过滤）
//   - recordType: 记录类型（0 不过滤）
//   - institutionID: 机构ID（0 不过滤，通过课卡→课程→机构过滤）
//   - offset/limit: 分页参数
func (m *AdminBusinessMapper) ListRecords(courseRecordID, recordType, institutionID int64, offset, limit int) ([]*AdminRecordRow, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	if courseRecordID != 0 {
		where += " AND r.course_record_id = ?"
		args = append(args, courseRecordID)
	}
	if recordType != 0 {
		where += " AND r.record_type = ?"
		args = append(args, recordType)
	}
	if institutionID != 0 {
		// 通过课卡→课程→机构过滤
		where += ` AND EXISTS (SELECT 1 FROM c_course_record cr JOIN c_course co ON cr.course_id = co.id WHERE cr.id = r.course_record_id AND co.institution_id = ?)`
		args = append(args, institutionID)
	}

	// 统计总数
	countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM c_record r %s`, where)
	var total int64
	if err := m.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("统计上课记录数失败: %w", err)
	}

	// 分页查询
	query := fmt.Sprintf(`
		SELECT r.id, r.course_record_id, r.record_time, r.record_remark, r.record_type,
		       r.record_change, r.operate_teacher_id, r.create_time, r.update_time
		FROM c_record r
		%s
		ORDER BY r.create_time DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, limit, offset)

	rows, err := m.db.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("查询上课记录列表失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminRecordRow
	for rows.Next() {
		row := &AdminRecordRow{}
		var (
			courseRecordID, recordType, recordChange, operateTeacherID sql.NullInt64
			recordTime, createTime, updateTime                        sql.NullTime
			recordRemark                                              sql.NullString
		)
		if err := rows.Scan(&row.ID, &courseRecordID, &recordTime, &recordRemark, &recordType, &recordChange, &operateTeacherID, &createTime, &updateTime); err != nil {
			return nil, 0, fmt.Errorf("扫描上课记录失败: %w", err)
		}
		if courseRecordID.Valid {
			row.CourseRecordID = courseRecordID.Int64
		}
		row.RecordTimeStr = formatTimeSQL(recordTime)
		row.RecordRemark = recordRemark.String
		if recordType.Valid {
			row.RecordType = recordType.Int64
		}
		if recordChange.Valid {
			row.RecordChange = recordChange.Int64
		}
		if operateTeacherID.Valid {
			row.OperateTeacherID = operateTeacherID.Int64
		}
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, total, nil
}

// InsertRecord 新增上课记录（对齐 Java insertRecord）
//
// 参数：
//   - courseRecordID: 课卡记录ID
//   - recordType: 记录类型
//   - recordChange: 课时变更数量
//   - recordTimeStr: 记录时间（"" 则使用 NOW()）
//   - recordRemark: 备注
//
// 返回：新记录ID
func (m *AdminBusinessMapper) InsertRecord(courseRecordID, recordType, recordChange int64, recordTimeStr, recordRemark string) (int64, error) {
	var timeArg interface{}
	if recordTimeStr != "" {
		timeArg = recordTimeStr
	} else {
		timeArg = nil // 使用数据库默认值（NOW()）
	}
	query := `INSERT INTO c_record (course_record_id, record_time, record_type, record_change, record_remark, create_time, update_time)
	          VALUES (?, COALESCE(?, NOW()), ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, courseRecordID, timeArg, recordType, recordChange, recordRemark)
	if err != nil {
		return 0, fmt.Errorf("新增上课记录失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取上课记录ID失败: %w", err)
	}
	return id, nil
}

// ============================================================
// 小程序菜单管理（c_menu + c_role_menu）
// ============================================================

// AdminMiniMenuRow 小程序菜单查询行（对齐 Java MenuVO + MiniMenuResponse）
//
// c_menu 表字段：id, menu_name, icon, icon_type, bg_color, path, sort_order, is_visible, create_time, update_time
type AdminMiniMenuRow struct {
	ID           int64   `json:"id"`           // 菜单ID
	MenuName     string  `json:"menuName"`     // 菜单名称
	Icon         string  `json:"icon"`         // 图标（名称或路径）
	IconType     int64   `json:"iconType"`     // 图标类型（0=内置, 1=路径）
	BgColor      string  `json:"bgColor"`      // 图标背景色
	Path         string  `json:"path"`         // 跳转路由路径
	SortOrder    int64   `json:"sortOrder"`    // 排序权值
	IsVisible    bool    `json:"isVisible"`    // 是否显示
	CreateTimeStr string `json:"createTimeStr"` // 创建时间字符串
	UpdateTimeStr string `json:"updateTimeStr"` // 更新时间字符串
	RoleIDs      []int64 `json:"roleIds"`      // 角色（权限）ID列表（3=家长端, 4=教师端）
}

// ListMiniMenus 查询所有小程序菜单（按 sort_order 升序）
//
// 对齐 Java listMiniMenus
// 注意：roleIDs 由 Service 层填充
func (m *AdminBusinessMapper) ListMiniMenus() ([]*AdminMiniMenuRow, error) {
	query := `SELECT id, menu_name, icon, icon_type, bg_color, path, sort_order, is_visible, create_time, update_time FROM c_menu ORDER BY sort_order ASC`
	rows, err := m.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询小程序菜单失败: %w", err)
	}
	defer rows.Close()

	var list []*AdminMiniMenuRow
	for rows.Next() {
		row := &AdminMiniMenuRow{RoleIDs: []int64{}}
		var (
			menuName, icon, bgColor, path sql.NullString
			iconType, sortOrder           sql.NullInt64
			isVisible                     sql.NullBool
			createTime, updateTime        sql.NullTime
		)
		if err := rows.Scan(&row.ID, &menuName, &icon, &iconType, &bgColor, &path, &sortOrder, &isVisible, &createTime, &updateTime); err != nil {
			return nil, fmt.Errorf("扫描菜单记录失败: %w", err)
		}
		row.MenuName = menuName.String
		row.Icon = icon.String
		if iconType.Valid {
			row.IconType = iconType.Int64
		}
		row.BgColor = bgColor.String
		row.Path = path.String
		if sortOrder.Valid {
			row.SortOrder = sortOrder.Int64
		}
		row.IsVisible = isVisible.Bool
		row.CreateTimeStr = formatTimeSQL(createTime)
		row.UpdateTimeStr = formatTimeSQL(updateTime)
		list = append(list, row)
	}
	return list, nil
}

// SelectMiniMenuByID 按ID查小程序菜单
func (m *AdminBusinessMapper) SelectMiniMenuByID(id int64) (*AdminMiniMenuRow, error) {
	query := `SELECT id, menu_name, icon, icon_type, bg_color, path, sort_order, is_visible, create_time, update_time FROM c_menu WHERE id = ?`
	row := m.db.QueryRow(query, id)
	r := &AdminMiniMenuRow{RoleIDs: []int64{}}
	var (
		menuName, icon, bgColor, path sql.NullString
		iconType, sortOrder           sql.NullInt64
		isVisible                     sql.NullBool
		createTime, updateTime        sql.NullTime
	)
	if err := row.Scan(&r.ID, &menuName, &icon, &iconType, &bgColor, &path, &sortOrder, &isVisible, &createTime, &updateTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询菜单失败: %w", err)
	}
	r.MenuName = menuName.String
	r.Icon = icon.String
	if iconType.Valid {
		r.IconType = iconType.Int64
	}
	r.BgColor = bgColor.String
	r.Path = path.String
	if sortOrder.Valid {
		r.SortOrder = sortOrder.Int64
	}
	r.IsVisible = isVisible.Bool
	r.CreateTimeStr = formatTimeSQL(createTime)
	r.UpdateTimeStr = formatTimeSQL(updateTime)
	return r, nil
}

// InsertMiniMenu 新增小程序菜单
//
// 参数：
//   - menuName: 菜单名称（必填）
//   - icon: 图标
//   - iconType: 图标类型（0=内置, 1=路径）
//   - bgColor: 背景色
//   - path: 跳转路径
//   - sortOrder: 排序权值
//   - isVisible: 是否显示
//
// 返回：新菜单ID
func (m *AdminBusinessMapper) InsertMiniMenu(menuName, icon string, iconType int64, bgColor, path string, sortOrder int64, isVisible bool) (int64, error) {
	query := `INSERT INTO c_menu (menu_name, icon, icon_type, bg_color, path, sort_order, is_visible, create_time, update_time)
	          VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())`
	result, err := m.db.Exec(query, menuName, icon, iconType, bgColor, path, sortOrder, isVisible)
	if err != nil {
		return 0, fmt.Errorf("新增小程序菜单失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取菜单ID失败: %w", err)
	}
	return id, nil
}

// UpdateMiniMenu 更新小程序菜单（动态更新非空字段）
//
// 参数：
//   - id: 菜单ID
//   - menuName: 菜单名称（"" 不更新）
//   - icon: 图标（"" 不更新）
//   - iconType: 图标类型（-1 不更新）
//   - bgColor: 背景色（"" 不更新）
//   - path: 跳转路径（"" 不更新）
//   - sortOrder: 排序权值（-1 不更新）
//   - isVisible: 是否显示（使用 *bool 传值，nil 不更新）
func (m *AdminBusinessMapper) UpdateMiniMenu(id int64, menuName, icon string, iconType int64, bgColor, path string, sortOrder int64, isVisible *bool) error {
	setParts := []string{"update_time = NOW()"}
	args := []interface{}{}
	if menuName != "" {
		setParts = append(setParts, "menu_name = ?")
		args = append(args, menuName)
	}
	if icon != "" {
		setParts = append(setParts, "icon = ?")
		args = append(args, icon)
	}
	if iconType != -1 {
		setParts = append(setParts, "icon_type = ?")
		args = append(args, iconType)
	}
	if bgColor != "" {
		setParts = append(setParts, "bg_color = ?")
		args = append(args, bgColor)
	}
	if path != "" {
		setParts = append(setParts, "path = ?")
		args = append(args, path)
	}
	if sortOrder != -1 {
		setParts = append(setParts, "sort_order = ?")
		args = append(args, sortOrder)
	}
	if isVisible != nil {
		setParts = append(setParts, "is_visible = ?")
		args = append(args, *isVisible)
	}

	query := fmt.Sprintf("UPDATE c_menu SET %s WHERE id = ?", strings.Join(setParts, ", "))
	args = append(args, id)
	if _, err := m.db.Exec(query, args...); err != nil {
		return fmt.Errorf("更新小程序菜单失败: %w", err)
	}
	return nil
}

// DeleteMiniMenu 删除小程序菜单
func (m *AdminBusinessMapper) DeleteMiniMenu(id int64) error {
	if _, err := m.db.Exec(`DELETE FROM c_menu WHERE id = ?`, id); err != nil {
		return fmt.Errorf("删除小程序菜单失败: %w", err)
	}
	return nil
}

// SelectRoleIDsByMenuID 查询菜单关联的角色（权限）ID列表
//
// 对齐 Java selectRoleIdsByMenuId
// 查询 c_role_menu 表（permission_id 即为 roleId）
func (m *AdminBusinessMapper) SelectRoleIDsByMenuID(menuID int64) ([]int64, error) {
	rows, err := m.db.Query(`SELECT permission_id FROM c_role_menu WHERE menu_id = ?`, menuID)
	if err != nil {
		return nil, fmt.Errorf("查询菜单角色ID失败: %w", err)
	}
	defer rows.Close()

	var roleIDs []int64
	for rows.Next() {
		var roleID int64
		if err := rows.Scan(&roleID); err != nil {
			return nil, fmt.Errorf("扫描角色ID失败: %w", err)
		}
		roleIDs = append(roleIDs, roleID)
	}
	if roleIDs == nil {
		roleIDs = []int64{}
	}
	return roleIDs, nil
}

// DeleteRoleMenuByMenuID 删除菜单的所有角色关联
//
// 对齐 Java deleteRoleMenuByMenuId
func (m *AdminBusinessMapper) DeleteRoleMenuByMenuID(menuID int64) error {
	if _, err := m.db.Exec(`DELETE FROM c_role_menu WHERE menu_id = ?`, menuID); err != nil {
		return fmt.Errorf("删除菜单角色关联失败: %w", err)
	}
	return nil
}

// InsertRoleMenu 新增菜单-角色关联
//
// 对齐 Java insertRoleMenu
// permission_id 即为 roleId（3=家长端, 4=教师端）
func (m *AdminBusinessMapper) InsertRoleMenu(roleID, menuID int64) error {
	if _, err := m.db.Exec(`INSERT INTO c_role_menu (permission_id, menu_id) VALUES (?, ?)`, roleID, menuID); err != nil {
		return fmt.Errorf("新增菜单角色关联失败: %w", err)
	}
	return nil
}

// ============================================================
// 用户认证（c_user_auth + c_user）相关方法
// ============================================================

// AdminUserAuthRow 用户认证查询行
type AdminUserAuthRow struct {
	ID            int64  `json:"id"`            // 主键
	UserID        int64  `json:"userId"`        // 用户ID
	Account       string `json:"account"`       // 账号
	Password      string `json:"-"`             // 密码（SM3 加盐哈希，JSON 不输出）
	Salt          string `json:"-"`             // 盐值（JSON 不输出）
	LastLoginTime string `json:"lastLoginTime"` // 最后登录时间字符串
}

// SelectUserAuthByTeacherID 按教师ID查用户认证信息
//
// 对齐 Java UserAuthMapper.selectAuthByTeacherId
// JOIN c_teacher + c_user 找到教师的 user_auth 记录（role_id=4）
func (m *AdminBusinessMapper) SelectUserAuthByTeacherID(teacherID int64) (*AdminUserAuthRow, error) {
	query := `
		SELECT ua.id, ua.user_id, ua.account, ua.password, ua.salt, ua.last_login_time
		FROM c_teacher t
		INNER JOIN c_user_auth ua ON ua.user_id = t.user_id AND ua.role_id = 4
		WHERE t.id = ?
		LIMIT 1
	`
	row := m.db.QueryRow(query, teacherID)
	r := &AdminUserAuthRow{}
	var (
		userID                   sql.NullInt64
		account, password, salt  sql.NullString
		lastLoginTime            sql.NullTime
	)
	if err := row.Scan(&r.ID, &userID, &account, &password, &salt, &lastLoginTime); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("查询教师账号信息失败: %w", err)
	}
	if userID.Valid {
		r.UserID = userID.Int64
	}
	r.Account = account.String
	r.Password = password.String
	r.Salt = salt.String
	r.LastLoginTime = formatTimeSQL(lastLoginTime)
	return r, nil
}

// ExistsUserAuthByInstitutionAndAccount 校验同机构同角色下账号是否已存在
//
// 对齐 Java UserAuthMapper.existsByInstitutionAndAccountAndRole
// 用于教师账号唯一性校验
//
// 参数：
//   - institutionID: 机构ID
//   - account: 账号
//   - roleID: 角色ID（教师=4）
//
// 返回：true=已存在
func (m *AdminBusinessMapper) ExistsUserAuthByInstitutionAndAccount(institutionID int64, account string, roleID int64) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM c_user_auth ua
			INNER JOIN c_user u ON ua.user_id = u.id
			WHERE u.institution_id = ? AND ua.account = ? AND ua.role_id = ?
		)
	`
	var exists bool
	if err := m.db.QueryRow(query, institutionID, account, roleID).Scan(&exists); err != nil {
		return false, fmt.Errorf("校验账号唯一性失败: %w", err)
	}
	return exists, nil
}

// InsertUserAuth 新增用户认证记录
//
// 对齐 Java userAuthMapper.insert
// 用于教师创建时同时创建账号
//
// 参数：
//   - userID: 用户ID
//   - roleID: 角色ID（教师=4）
//   - account: 账号
//   - password: 密码（SM3 加盐哈希）
//   - salt: 盐值
//
// 返回：新认证记录ID
func (m *AdminBusinessMapper) InsertUserAuth(userID, roleID int64, account, password, salt string) (int64, error) {
	query := `INSERT INTO c_user_auth (user_id, role_id, account, password, salt, last_login_time) VALUES (?, ?, ?, ?, ?, NULL)`
	result, err := m.db.Exec(query, userID, roleID, account, password, salt)
	if err != nil {
		return 0, fmt.Errorf("新增用户认证失败: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取认证ID失败: %w", err)
	}
	return id, nil
}

// UpdateUserAuthAccount 更新用户认证账号
//
// 对齐 Java userAuthMapper.updateById（仅更新 account）
func (m *AdminBusinessMapper) UpdateUserAuthAccount(authID int64, account string) error {
	if _, err := m.db.Exec(`UPDATE c_user_auth SET account = ? WHERE id = ?`, account, authID); err != nil {
		return fmt.Errorf("更新账号失败: %w", err)
	}
	return nil
}

// UpdateUserAuthPassword 更新用户认证密码
//
// 对齐 Java userAuthMapper.updateById（更新 password + salt）
func (m *AdminBusinessMapper) UpdateUserAuthPassword(authID int64, password, salt string) error {
	if _, err := m.db.Exec(`UPDATE c_user_auth SET password = ?, salt = ? WHERE id = ?`, password, salt, authID); err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return nil
}

// SelectUserInstitutionID 按用户ID查机构ID
//
// 用于教师账号更新时校验同机构账号唯一性
func (m *AdminBusinessMapper) SelectUserInstitutionID(userID int64) (int64, error) {
	var instID sql.NullInt64
	if err := m.db.QueryRow(`SELECT institution_id FROM c_user WHERE id = ?`, userID).Scan(&instID); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("查询用户机构ID失败: %w", err)
	}
	if instID.Valid {
		return instID.Int64, nil
	}
	return 0, nil
}

// UpdateTeacherInstitutionAdmin 更新教师机构管理员标识
//
// 对齐 Java teacherMapper.updateById（仅更新 is_institution_admin 字段）
//
// 参数：
//   - teacherID: 教师ID
//   - isInstitutionAdmin: 是否机构管理员
func (m *AdminBusinessMapper) UpdateTeacherInstitutionAdmin(teacherID int64, isInstitutionAdmin bool) error {
	if _, err := m.db.Exec(`UPDATE c_teacher SET is_institution_admin = ? WHERE id = ?`, isInstitutionAdmin, teacherID); err != nil {
		return fmt.Errorf("更新机构管理员标识失败: %w", err)
	}
	return nil
}

// ============================================================
// 辅助函数
// ============================================================

// formatTimeSQL 格式化 sql.NullTime 为 "yyyy-MM-dd HH:mm:ss" 字符串
//
// 无效时间返回空字符串
func formatTimeSQL(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02 15:04:05")
}

// formatDateSQL 格式化 sql.NullTime 为 "yyyy-MM-dd" 字符串
//
// 用于 birth/start_date/end_date 等日期字段
func formatDateSQL(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("2006-01-02")
}

// formatTimePart 格式化 sql.NullTime 为 "15:04:05" 字符串
//
// 用于 start_time/end_time 等时间字段
func formatTimePart(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format("15:04:05")
}
