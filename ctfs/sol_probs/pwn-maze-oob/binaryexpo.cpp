#include <iostream>
#include <cstdlib>

using namespace std;

// We bundle the variables together to guarantee their layout in memory.
// The map is 2700 bytes (30 * 90). The win_flag sits directly after it.
struct GameState {
    char map[30][90];
    int win_flag;
};

void win() {
    cout << "\n[+] ACCESS GRANTED. PANTHEON{0ut_0f_b0unds_https://crypto-sphinx-oracle.onrender.com}" << endl;
    cout << "[+] Next Trial: https://crypto-sphinx-oracle.onrender.com" << endl;
    exit(0);
}

int main() {
    // Disable buffering so output works smoothly over a network socket
    setvbuf(stdout, NULL, _IONBF, 0);
    
    GameState state;
    state.win_flag = 0;
    
    int player_x = 11;
    int player_y = 46;
    int end_x = 29;
    int end_y = 89;

    // Initialize the map with dots
    for(int i = 0; i < 30; i++) {
        for(int j = 0; j < 90; j++) {
            state.map[i][j] = '.';
        }
    }
    state.map[end_x][end_y] = 'X';

    while(true) {
        cout << "\033[2J\033[1;1H"; // Clear screen
        cout << "Player position: " << player_x << " " << player_y << endl;
        cout << "End tile position: " << end_x << " " << end_y << endl;
        
        // Render the map
        for(int i = 0; i < 30; i++) {
            for(int j = 0; j < 90; j++) {
                if(i == player_x && j == player_y) {
                    cout << '@';
                } else {
                    cout << state.map[i][j];
                }
            }
            cout << endl;
        }

        char move;
        cin >> move;

        // THE VULNERABILITY: No bounds checking on the player's coordinates!
        if(move == 'w') player_x--;
        else if(move == 's') player_x++;
        else if(move == 'a') player_y--;
        else if(move == 'd') player_y++;
        else if(move == 'p') {
            // "Secret command": Drops a breadcrumb at the current location
            state.map[player_x][player_y] = 'P'; 
        }

        // Win condition check
        if(player_x == end_x && player_y == end_y) {
            if(state.win_flag != 0) {
                win();
            } else {
                cout << "\nYou reached the exit, but you don't have the secret key!" << endl;
                exit(0);
            }
        }
    }
    return 0;
}