/*
 * Create: Sat Jul 27 21:52:53 2024
 */
#include <stdio.h>
#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/syscall.h>

pthread_mutex_t lock;
pthread_cond_t cond;


pid_t gettid(void)
{
    return syscall(SYS_gettid);
}


void* thread1_func(void* arg) {

    sleep(1);
    printf("tid:%d\n", gettid());
    pthread_mutex_lock(&lock);
    pthread_cond_signal(&cond);
    pthread_mutex_unlock(&lock);

    return NULL;
}

void* thread2_func(void* arg) {

    printf("tid:%d\n", gettid());
    pthread_mutex_lock(&lock);
    pthread_cond_wait(&cond, &lock);
    pthread_mutex_unlock(&lock);

    return NULL;
}

int main() {

    sleep(15);
    pthread_t thread1, thread2;

    pthread_mutex_init(&lock, NULL);
    pthread_cond_init(&cond, NULL);

    pthread_create(&thread1, NULL, thread1_func, NULL);
    pthread_create(&thread2, NULL, thread2_func, NULL);

    pthread_join(thread1, NULL);
    pthread_join(thread2, NULL);

    pthread_mutex_destroy(&lock);
    pthread_cond_destroy(&cond);

    return 0;
}
