/**
 * Tugas yang harus jalan selagi token akses masih ada, sebelum sesi dihapus
 * (misalnya menghapus token push perangkat di server).
 */
type Task = () => Promise<unknown>;

const tasks = new Set<Task>();
const TASK_TIMEOUT_MS = 3_000;

/** Mendaftarkan tugas; mengembalikan fungsi untuk melepasnya. */
export function addSignOutTask(task: Task) {
  tasks.add(task);
  return () => {
    tasks.delete(task);
  };
}

/** Menjalankan semua tugas paralel. Gagal atau lambat tidak menghalangi logout. */
export async function runSignOutTasks() {
  const timeout = new Promise((resolve) => setTimeout(resolve, TASK_TIMEOUT_MS));
  await Promise.race([Promise.allSettled([...tasks].map((task) => task())), timeout]);
}
