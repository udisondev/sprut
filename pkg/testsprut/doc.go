// Package testsprut предоставляет тестовое окружение для интеграционных тестов Sprut.
//
// Пакет позволяет в одну строку поднять полноценное окружение:
// NATS контейнер через testcontainers + Sprut сервер программно.
//
// Использование в тестах:
//
//	func TestIntegration(t *testing.T) {
//	    ctx := context.Background()
//
//	    env, err := testsprut.Start(ctx)
//	    require.NoError(t, err)
//	    defer env.Close(ctx)
//
//	    keys, _ := identity.Generate()
//	    client, _ := env.NewClient(ctx, keys)
//	    defer client.Close()
//
//	    // Тестируем свой код
//	}
//
// Использование в другом проекте:
//
//	// go.mod
//	require github.com/udisondev/sprut v0.x.x
//
//	// myproject/integration_test.go
//	import "github.com/udisondev/sprut/pkg/testsprut"
package testsprut
