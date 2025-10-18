package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	storage "EffectiveMobile/internal/storage"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	pgC  *tcpg.PostgresContainer
	dsn  string
	pool *pgxpool.Pool
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	c, err := tcpg.Run(ctx,
		"docker.io/postgres:16-alpine",
		tcpg.WithDatabase("testdb"),
		tcpg.WithUsername("testuser"),
		tcpg.WithPassword("testpass"),
		testcontainers.WithExposedPorts("5444:5432/tcp"),
		testcontainers.WithTmpfs(map[string]string{
			"/var/lib/postgresql/data": "rw",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp").WithStartupTimeout(60*time.Second),
		),
	)

	if err != nil {
		panic(err)
	}
	pgC = c

	host, err := pgC.Host(ctx)
	if err != nil {
		panic(err)
	}
	mp, err := pgC.MappedPort(ctx, "5432/tcp")
	if err != nil {
		panic(err)
	}

	dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		"testuser", "testpass", host, mp.Port(), "testdb")

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		panic(err)
	}
	if err := applyUps(sqlDB, findMigrationsDirMust()); err != nil {
		panic(err)
	}
	_ = sqlDB.Close()

	pool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		panic(err)
	}

	code := m.Run()

	pool.Close()
	_ = pgC.Terminate(ctx)
	os.Exit(code)
}

func findMigrationsDirMust() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	dir := wd
	for {
		p := filepath.Join(dir, "migrations")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return p
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("migrations dir not found")
		}
		dir = parent
	}
}

// applyUps выполняет ВСЕ *.up.sql по порядку
func applyUps(db *sql.DB, dir string) error {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return errors.New("no *.up.sql migrations found")
	}
	sort.Slice(files, func(i, j int) bool { return files[i] < files[j] })

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, f := range files {
		sqlBytes, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("apply %s: %w", filepath.Base(f), err)
		}
	}
	return tx.Commit()
}

func newStorage() *Storage {
	return &Storage{db: pool}
}

func m1(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func Test_AddService(t *testing.T) {
	s := newStorage()
	ctx := context.Background()
	cases := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"ok", "Yandex Plus", nil},
		{"trim_spaces_ok", "   Netflix   ", nil},
		{"empty", "", errors.New("empty service name")},
		{"whitespace_only", "   ", errors.New("empty service name")},
		{"duplicate", "Yandex Plus", storage.ErrServiceNameExists},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			id, err := s.AddService(ctx, tc.input)
			if tc.wantErr != nil {
				require.Error(t, err)
				switch tc.wantErr {
				case storage.ErrServiceNameExists:
					assert.ErrorIs(t, err, storage.ErrServiceNameExists)
				default:
					assert.EqualError(t, err, tc.wantErr.Error())
				}
				return
			}
			require.NoError(t, err)
			assert.Greater(t, id, 0)
		})

	}
}

func Test_GetServiceName(t *testing.T) {
	s := newStorage()
	ctx := context.Background()
	id, err := s.AddService(ctx, "YouTube")
	require.NoError(t, err)
	cases := []struct {
		name     string
		inID     int
		wantName string
		wantErr  error
	}{
		{"ok", id, "YouTube", nil},
		{"not_found_big_id", 99999999, "", storage.ErrServiceNotFound},
		{"not_found_zero_id", 0, "", storage.ErrServiceNotFound},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.GetServiceName(ctx, tc.inID)
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantName, got)
		})
	}
}

func Test_GetServiceID(t *testing.T) {
	s := newStorage()
	ctx := context.Background()
	_, err := s.AddService(ctx, "Yandex Plus 2")
	require.NoError(t, err)
	cases := []struct {
		name      string
		inName    string
		wantPosID bool
		wantErr   error
	}{
		{"ok", "Yandex Plus 2", true, nil},
		{"not_found", "Unknown Service", false, storage.ErrServiceNotFound},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := s.GetServiceID(ctx, tc.inName)
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Greater(t, got, 0)
		})
	}
}

func Test_GetOrCreateServiceID(t *testing.T) {
	s := newStorage()
	ctx := context.Background()
	existingID, err := s.AddService(ctx, "Twitch")
	require.NoError(t, err)
	cases := []struct {
		name   string
		inName string
		expect func(t *testing.T, id int, err error)
	}{
		{
			"existing_returns_old_id",
			"Twitch",
			func(t *testing.T, id int, err error) {
				require.NoError(t, err)
				assert.Equal(t, existingID, id)
			},
		},
		{
			"creates_new",
			"Spotify",
			func(t *testing.T, id int, err error) {
				require.NoError(t, err)
				assert.Greater(t, id, 0)
				assert.NotEqual(t, existingID, id)
			},
		},
		{
			"validation_empty",
			"   ",
			func(t *testing.T, id int, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "empty service name")
			},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			id, err := s.GetOrCreateServiceID(ctx, tc.inName)
			tc.expect(t, id, err)
		})
	}
}

func Test_DeleteService(t *testing.T) {
	s := newStorage()
	ctx := context.Background()
	toDelete, err := s.AddService(ctx, "Del A")
	require.NoError(t, err)
	withFK, err := s.AddService(ctx, "Del FK")
	require.NoError(t, err)
	u := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO subscription(user_id, service_id, price_rub, start_date) VALUES ($1,$2,$3,$4)`, u, withFK, 100, m1(time.Now().UTC()))
	require.NoError(t, err)
	cases := []struct {
		name    string
		inID    int
		wantErr error
		after   func(t *testing.T)
	}{
		{
			"ok",
			toDelete,
			nil,
			func(t *testing.T) {
				_, err := s.GetServiceName(ctx, toDelete)
				require.Error(t, err)
				assert.ErrorIs(t, err, storage.ErrServiceNotFound)
			},
		},
		{"not_found", 42424242, storage.ErrServiceNotFound, func(t *testing.T) {}},
		{"fk_in_use", withFK, storage.ErrServiceInUse, func(t *testing.T) {}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := s.DeleteService(ctx, tc.inID)
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			tc.after(t)
		})
	}
}
