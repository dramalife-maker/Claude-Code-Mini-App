package ws

import (
	"context"
	"testing"
)

// 重現「權限被拒後使用者允許重跑」的競態：A 被 taskCancel 取代成 B 後，
// A 的 goroutine 才姍姍來遲執行 defer taskEnd。修正前 taskEnd 只認 sessionID，
// 會把剛登記的 B 誤刪，導致 taskIsActive 提早回報 false（B 明明還在跑）。
func TestTaskEnd_DoesNotDeleteNewerTask(t *testing.T) {
	const sid = "s1"
	_, cancelA := context.WithCancel(context.Background())
	taskStart(sid, cancelA, 1) // A: msgID=1

	// 模擬 allow_once：taskCancel(A) 後立刻 taskStart(B)。
	taskCancel(sid)
	_, cancelB := context.WithCancel(context.Background())
	taskStart(sid, cancelB, 2) // B: msgID=2

	// A 的 goroutine 這時才跑到 defer taskEnd，帶的是自己的舊 msgID。
	taskEnd(sid, 1)

	if !taskIsActive(sid) {
		t.Fatal("B 應仍視為進行中，taskEnd(舊 msgID) 不該刪掉 B 的登記")
	}

	taskEnd(sid, 2)
	if taskIsActive(sid) {
		t.Fatal("B 用自己的 msgID 收尾後應清除登記")
	}
}
