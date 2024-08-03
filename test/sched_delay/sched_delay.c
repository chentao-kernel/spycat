/*
 * Create: Tue Jul 16 19:53:43 2024
 */
/*
 * Create: Tue Jul 16 19:35:30 2024
 */
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sched.h>
#include <sys/syscall.h>

#define NUM_THREADS 8
#define SLEEP_TIME 20

// gcc -pthread -o sched_delay sched_delay.c
//
pid_t get_tid(void)
{
        return syscall(SYS_gettid);
}

void* sleepy_thread(void* arg) {

    printf("sleepy thread:%d\n", get_tid());
    // Sleep for a specified time
    sleep(SLEEP_TIME);

    // After sleep, keep yielding
    while (1) {
        sched_yield();
    }

    pthread_exit(NULL);
}

void* working_thread(void* arg) {
    // Simulate work by keeping the CPU busy
    printf("working thread:%d\n", get_tid());
    while (1) {
        // Perform some work, e.g., a busy-wait loop
        volatile int i;
        for (i = 0; i < 1000000; ++i);
    }

    pthread_exit(NULL);
}

int main(int argc, char* argv[]) {
    pthread_t threads[NUM_THREADS + 1];
    int rc;
    long t;

    // Create sleepy threads
    for (t = 0; t < NUM_THREADS; ++t) {
        rc = pthread_create(&threads[t], NULL, sleepy_thread, (void*)t);
        if (rc) {
            printf("ERROR; return code from pthread_create() is %d\n", rc);
            exit(-1);
        }
    }

    // Create working thread
    rc = pthread_create(&threads[NUM_THREADS], NULL, working_thread, (void*)t);
    if (rc) {
        printf("ERROR; return code from pthread_create() is %d\n", rc);
        exit(-1);
    }

    // Wait for the threads to complete (they won't, so this is just for completeness)
    for (t = 0; t < NUM_THREADS + 1; ++t) {
        pthread_join(threads[t], NULL);
    }

    pthread_exit(NULL);
}
