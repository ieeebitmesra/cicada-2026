import sys
import string
import time

# The secret flag
FLAG = "PANTHEON{Cr4ck1ng_Th3_0lymp14n_0r4cl3_c0mpl3t3d}"
SECRET_OFFERING = "OLYMPIANNECTAR"
VIGENERE_KEY = "ZEUS"

def encrypt_offering(plaintext, key):
    """Encrypts text using a Vigenere cipher with the given key."""
    ciphertext = ""
    key_index = 0
    for char in plaintext.upper():
        if char in string.ascii_uppercase:
            shift = ord(key[key_index % len(key)]) - 65
            encrypted_char = chr((ord(char) - 65 + shift) % 26 + 65)
            ciphertext += encrypted_char
            key_index += 1
        else:
            ciphertext += char
    return ciphertext

ENCRYPTED_OFFERING = encrypt_offering(SECRET_OFFERING, VIGENERE_KEY)

# --- ASCII ART FUNCTIONS ---

def print_sphinx_idle():
    print(r"""
         /\_/\
        ( o.o )
         > ^ <
        /  -  \
       /       \
      (   | |   )
       \  | |  /
        `-\_m_m_/
    """)

def print_sphinx_tasting():
    print(r"""
         /\_/\        _ 
        ( -.- )     _(_)
         > ^ <     (_) \ 
        /  o  \      | | 
       /  \_/  \    /   \
      (   | |   )   \___/
       \  | |  /    __|__
        `-\_m_m_/   `---`
    """)

def print_sphinx_angry():
    print(r"""
         /\_/\
        ( >_< )
         > V <
        /  0  \
       /       \
      (   | |   )
       \  | |  /
        `-\_m_m_/
    """)

# --- MAIN LOGIC ---

def main():
    print("*" * 50)
    print("***                   Part 1                   ***")
    print("***      The Mystery of the CLONED DEMIGOD     ***")
    print("*" * 50)
    print("\nHalt! I am the Sphinx of the Pantheon. Hades has sent clones")
    print("to infiltrate Mount Olympus. You look like a true Demigod,")
    print("but I must be sure.")
    
    chances = 4

    while chances > 0:
        print(f"\nRemember, my encrypted offering is:  {ENCRYPTED_OFFERING}")
        print("Commands: (g)uess my offering or (e)ncrypt an offering")
        choice = input("What would you like to do?\n> ").strip().lower()

        if choice == 'e':
            offering_to_encrypt = input("\nWhat offering would you like to encrypt?\n> ")
            print("\nLet me consult the Oracle... If you give me that, it encrypts to:")
            print(encrypt_offering(offering_to_encrypt, VIGENERE_KEY))
            
            chances -= 1
            if chances > 0:
                print(f"\nI grow weary of these tests. You have {chances} chances left!")
            
        elif choice == 'g':
            print_sphinx_idle()
            print("ROAR ROAR ROAR\n")
            print("Are you the true Demigod? Are you ready to GUESS...MY...OFFERING?")
            guess = input(f"Remember, this is my encrypted offering: {ENCRYPTED_OFFERING}\nSo...what's my offering?\n> ").strip().upper()
            
            # The Dramatic Tasting Sequence!
            print("\nsip...")
            print_sphinx_tasting()
            time.sleep(1)
            
            print("sip...")
            print_sphinx_tasting()
            time.sleep(1)
            
            print("SIP.............\n")
            print_sphinx_tasting()
            time.sleep(1.5)
            
            if guess == SECRET_OFFERING:
                print_sphinx_idle()
                print("Mmm... That IS my offering! The magic is real.")
                print("You are no clone! Welcome to Olympus, Demigod.")
                print(f"Here is your divine reward: {FLAG}\n")
                sys.exit(0)
            else:
                print_sphinx_angry()
                print("BLEHHHHH yuck. THAT'S not my offering. GET OUT CLONE!!!1111!!!1!!!\n")
                sys.exit(0)
        else:
            print("Speak clearly! I do not understand that command.")

    print("\nYou have exhausted my patience! The gates remain closed.")

if __name__ == "__main__":
    main()
