/*
 * Create: Mon Feb 19 20:13:13 2024
 */

#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <time.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/syscall.h> /*必须引用这个文件 */

#define N_THREADS 5

pid_t gettid(void)
{
	return syscall(SYS_gettid);
}

pthread_mutex_t lock;

void test_lock1(pthread_mutex_t *lock)
{
	int a = 0;
	a++;
	printf("%d\n", a);
	pthread_mutex_lock(lock);
}

void test_lock2(pthread_mutex_t *lock)
{
	test_lock1(lock);
}

void test_unlock1(pthread_mutex_t *lock)
{
	int a = 0;
	a++;
	printf("%d\n", a);
	pthread_mutex_unlock(lock);
}

void test_unlock2(pthread_mutex_t *lock)
{
	test_unlock1(lock);
}

void my_handle(void *arg)
{
	struct timespec time;
	int tid;

	tid = gettid();

	sleep(1);
	for (int i = 0; i < 3; i++) {
		clock_gettime(CLOCK_MONOTONIC, &time);
		printf("%lu.%lu, tid:%lu, pid:%d, require mutex:%x\n",
		       time.tv_sec, time.tv_nsec, tid, getpid(), &lock);
		test_lock2(&lock);
		clock_gettime(CLOCK_MONOTONIC, &time);
		printf("%lu.%lu, tid:%d, pid:%d, get mutex:%x\n", time.tv_sec,
		       time.tv_nsec, tid, getpid(), &lock);
		sleep(1);
		clock_gettime(CLOCK_MONOTONIC, &time);
		printf("%lu.%lu, tid:%d, pid:%d, put mutex:%x\n", time.tv_sec,
		       time.tv_nsec, tid, getpid(), &lock);
		test_unlock2(&lock);
	}
	pthread_exit(NULL);
}

void *thread_handle(void *arg)
{
	my_handle(arg);
}

int main(void)
{
	pthread_t tid[N_THREADS];

	sleep(10);
	pthread_mutex_init(&lock, NULL);
	printf("Start Process:%d\n", getpid());
	for (int i = 0; i < N_THREADS; i++) {
		pthread_create(&tid[i], NULL, &thread_handle, NULL);
	}
	for (int i = 0; i < N_THREADS; i++) {
		pthread_join(tid[i], NULL);
	}
	test_lock2(&lock);
	sleep(2);
	test_unlock2(&lock);
	pthread_mutex_destroy(&lock);
	return 0;
}
