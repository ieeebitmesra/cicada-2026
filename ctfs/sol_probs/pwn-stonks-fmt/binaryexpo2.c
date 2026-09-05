#include <stdio.h>
#include <stdlib.h>
#include <string.h>

void buy_stonks() {
    char api_token[300];

    // The secret flag: "PANTHEON{f0rm4t_5tr1ng_https://pwn-maze-oob-1.onrender.com}"
    // Base64 encoded: "UEFOVEhFT057ZjBybTR0XzV0cjFuZ19odHRwczovL3B3bi1tYXplLW9vYi0xLm9ucmVuZGVyLmNvbX0="
    // By declaring this as a local array, it gets pushed onto the stack.
    char secret_b64[] = "UEFOVEhFT057ZjBybTR0XzV0cjFuZ19odHRwczovL3B3bi1tYXplLW9vYi0xLm9ucmVuZGVyLmNvbX0="; 

    printf("\nUsing patented AI algorithms to find the best stonks...\n");
    printf("Please enter your API token to authorize the trade:\n> ");

    // Read user input safely...
    scanf("%299s", api_token);

    printf("\nProcessing trade for token:\n");

    // THE VULNERABILITY: 
    // The input is evaluated as a format string. 
    // An attacker can input %p to read pointers (and local variables) directly off the stack!
    printf(api_token);
    printf("\n"); 
}

void view_portfolio() {
    printf("\nYour Portfolio:\n");
    printf("- 100 shares of GME\n");
    printf("- 50 shares of AMC\n\n");
}

int main() {
    // Disable buffering so output works smoothly over nc (socat)
    setvbuf(stdout, NULL, _IONBF, 0);
    int choice;

    printf("Welcome back to the trading app!\n\n");

    while(1) {
        printf("What would you like to do?\n");
        printf("1) Buy some stonks!\n");
        printf("2) View my portfolio\n");
        printf("3) Exit\n> ");

        if (scanf("%d", &choice) != 1) {
            printf("Invalid input. Exiting.\n");
            exit(1);
        }

        if (choice == 1) {
            buy_stonks();
        } else if (choice == 2) {
            view_portfolio();
        } else {
            printf("Goodbye!\n");
            exit(0);
        }
    }
    return 0;
}
