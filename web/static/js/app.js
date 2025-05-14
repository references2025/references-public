document.addEventListener('DOMContentLoaded', function () {
    const gameContainer = document.getElementById('game-container');

    if (!gameContainer) {
        console.error("Fatal: game-container element not found.");
        return;
    }
    function generatePlayerID() {
        const playerIDKey = 'references-player-id';
        let playerID = localStorage.getItem(playerIDKey);
        
        if (!playerID) {
            // Generate a random ID
            playerID = 'player-' + Math.random().toString(36).substring(2, 15) + 
                      Math.random().toString(36).substring(2, 15);
            localStorage.setItem(playerIDKey, playerID);
        }
        
        return playerID;
    }
    const gameId = String(gameContainer.dataset.gameId);
    const guessesLeftElem = document.getElementById('guesses-left');
    const hintsUsedElem = document.getElementById('hints-used');
    const wordDisplayElem = document.getElementById('word-display');
    const guessInput = document.getElementById('guess-input');
    const guessButton = document.getElementById('guess-button');
    const gameResults = document.getElementById('game-results');
    const hintBoxes = document.querySelectorAll('.hint-box');
    const playerID = generatePlayerID();


    const gameStateKey = `references-state-${gameId}`;
    const hintsStateKey = `references-hints-${gameId}`;

    let remainingGuesses = 4;
    let revealedHintsData = {};
    let usedHintsCount = 0;
    let wordLength = 0;

    // Get the word length from the masked word
    if (wordDisplayElem) {
        const maskedWord = wordDisplayElem.textContent.trim();
        wordLength = maskedWord.replace(/\s/g, '').length;
        
        // Hide the word display element as we'll use OTP input instead
        wordDisplayElem.style.display = 'none';
    }

    function loadGameState() {
        const storedState = localStorage.getItem(gameStateKey);
        if (storedState) {
            try {
                const state = JSON.parse(storedState);
                remainingGuesses = (typeof state.remainingGuesses === 'number' && state.remainingGuesses >= 0) ? state.remainingGuesses : 4;
            } catch (e) {
                console.error("Error parsing game state from localStorage:", e);
                localStorage.removeItem(gameStateKey);
            }
        }

        const storedHints = localStorage.getItem(hintsStateKey);
        if (storedHints) {
            try {
                revealedHintsData = JSON.parse(storedHints);
                if (typeof revealedHintsData !== 'object' || revealedHintsData === null) {
                    revealedHintsData = {};
                }
            } catch (e) {
                console.error("Error parsing hints state from localStorage:", e);
                localStorage.removeItem(hintsStateKey);
                revealedHintsData = {};
            }
        }

        usedHintsCount = Object.keys(revealedHintsData).length;

        if (guessesLeftElem) guessesLeftElem.textContent = remainingGuesses;
        if (hintsUsedElem) hintsUsedElem.textContent = usedHintsCount;

        hintBoxes.forEach(box => {
            const category = box.dataset.category;
            if (category && revealedHintsData[category]) {
                const hintContent = box.querySelector('.hint-content');
                if (hintContent) {
                    hintContent.textContent = revealedHintsData[category].hint;
                    box.classList.add('hint-revealed');
                }
            }
        });
    }

    function saveGameState(maskedWord) {
        const state = {
            remainingGuesses: remainingGuesses,
        };
        if (maskedWord) {
            state.maskedWord = maskedWord;
        }
        localStorage.setItem(gameStateKey, JSON.stringify(state));

        if (guessesLeftElem) guessesLeftElem.textContent = remainingGuesses;
    }

    function saveHintsState() {
        localStorage.setItem(hintsStateKey, JSON.stringify(revealedHintsData));

        usedHintsCount = Object.keys(revealedHintsData).length;
        if (hintsUsedElem) hintsUsedElem.textContent = usedHintsCount;
    }

    function clearGameState() {
        localStorage.removeItem(gameStateKey);
        localStorage.removeItem(hintsStateKey);
        console.log(`Cleared game state for game ID: ${gameId}`);
    }

    loadGameState();

    function trackGameEvent(type, value, correct = false) {
        const eventsKey = `references-events-${gameId}`;
        let events = [];

        try {
            const storedEvents = localStorage.getItem(eventsKey);
            if (storedEvents) {
                events = JSON.parse(storedEvents);
            }
        } catch (e) {
            console.error("Error parsing events:", e);
        }

        events.push({ type, value, correct });
        localStorage.setItem(eventsKey, JSON.stringify(events));
    }

    // This function handles the guess submission
    function handleGuessSubmission(guess) {
        if (remainingGuesses <= 0) return;
        
        if (!guess) {
            gameResults.textContent = "Please enter a guess.";
            return;
        }

        // Disable inputs during submission
        const submitButton = document.getElementById('guess-button');
        submitButton.disabled = true;
        
        // If we're using the OTP input, we need to disable all the individual inputs
        const otpInputs = document.querySelectorAll('.otp-input');
        otpInputs.forEach(input => {
            input.disabled = true;
        });
        
        gameResults.textContent = 'Checking...';

        fetch('/guess', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: `guess=${encodeURIComponent(guess)}&playerID=${encodeURIComponent(generatePlayerID())}`
        })
        .then(response => {
            if (!response.ok) return response.json().then(errData => { throw new Error(errData.error || `HTTP error ${response.status}`); });
            return response.json();
        })
        .then(data => {
            trackGameEvent('guess', guess, data.correct);
            gameResults.textContent = '';

            saveGameState(data.maskedWord);

            if (data.correct) {
                const guessesTaken = 4 - remainingGuesses + 1;
                const finalHintsCount = Object.keys(revealedHintsData).length;
                console.log(`Redirecting to success. Guesses: ${guessesTaken}, Hints: ${finalHintsCount}`);
                window.location.href = `/success?word=${encodeURIComponent(data.word)}&guesses=${guessesTaken}&hints=${finalHintsCount}&gameId=${encodeURIComponent(gameId)}`;
                return;
            }

            remainingGuesses--;
            saveGameState(data.maskedWord);

            if (remainingGuesses <= 0) {
                const finalHintsCount = Object.keys(revealedHintsData).length;
                const guessesTaken = 4;
                console.log(`Redirecting to maybe-tomorrow. Guesses: ${guessesTaken}, Hints: ${finalHintsCount}`);
                window.location.href = `/maybe-tomorrow?word=${encodeURIComponent(data.word)}&guesses=${guessesTaken}&hints=${finalHintsCount}&gameId=${encodeURIComponent(gameId)}`;
                return;
            }

            gameResults.textContent = 'Incorrect guess.';
        })
        .catch(error => {
            console.error('Guess Error:', error);
            gameResults.textContent = `Error: ${error.message || 'Could not process guess.'}`;
        })
        .finally(() => {
            if (remainingGuesses > 0) {
                // Re-enable inputs if we still have guesses left
                submitButton.disabled = false;
                
                otpInputs.forEach(input => {
                    input.disabled = false;
                });
                
                // Clear and focus the first input in OTP mode or the main input otherwise
                if (otpInputs.length > 0) {
                    otpInputs.forEach(input => input.value = '');
                    otpInputs[0].focus();
                } else if (guessInput) {
                    guessInput.value = '';
                    guessInput.focus();
                }
            }
        });
    }

    // Attach event handler to original guess button if it exists
    if (guessButton && guessInput) {
        guessButton.addEventListener('click', function (event) {
            event.preventDefault();
            const guess = guessInput.value.trim();
            handleGuessSubmission(guess);
        });

        guessInput.addEventListener('keypress', function (event) {
            if (event.key === 'Enter') {
                event.preventDefault();
                guessButton.click();
            }
        });
    }

    hintBoxes.forEach(box => {
        const category = box.dataset.category;

        box.addEventListener('click', function () {
            if (this.classList.contains('hint-revealed') || this.classList.contains('hint-loading')) {
                return;
            }

            const categoryName = this.dataset.category;
            const hintContent = this.querySelector('.hint-content');

            this.classList.add('hint-loading');
            hintContent.textContent = 'Loading...';

            fetch('/hint', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: `category=${encodeURIComponent(categoryName)}&playerID=${encodeURIComponent(generatePlayerID())}`
            })
            .then(response => {
                if (!response.ok) {
                    throw new Error(`HTTP error ${response.status}`);
                }
                return response.json();
            })
            .then(data => {
                trackGameEvent('hint', categoryName);
                hintContent.textContent = data.hint;

                this.classList.remove('hint-loading');
                this.classList.add('hint-revealed');

                revealedHintsData[categoryName] = {
                    hint: data.hint,
                    emoji: data.emoji || ''
                };

                usedHintsCount = Object.keys(revealedHintsData).length;
                if (hintsUsedElem) hintsUsedElem.textContent = usedHintsCount;
                saveHintsState();
            })
            .catch(error => {
                console.error('Hint Error:', error);

                this.classList.remove('hint-loading');
                hintContent.textContent = '';
                if (gameResults) gameResults.textContent = `Error loading hint: ${error.message}`;
            });
        });
    });

    function setupOTPInput() {
        const wordDisplayElem = document.getElementById('word-display');
        const gameStatus = document.getElementById('game-status');
        const gameResults = document.getElementById('game-results');
        const originalGuessInput = document.getElementById('guess-input');
        const originalGuessButton = document.getElementById('guess-button');
        
        // Only proceed if we have the required elements
        if (!wordDisplayElem || !gameStatus) {
            console.error("Required elements not found for OTP setup");
            return;
        }
        
        // Get word length from the masked word
        const maskedWord = wordDisplayElem.textContent.trim();
        const wordLength = maskedWord.replace(/\s/g, '').length;
        
        // Create OTP container
        const otpContainer = document.createElement('div');
        otpContainer.className = 'otp-container';
        otpContainer.style.display = 'flex';
        otpContainer.style.flexDirection = 'row';
        otpContainer.style.justifyContent = 'center';
        otpContainer.style.gap = '8px';
        otpContainer.style.width = '100%';
        otpContainer.style.margin = '25px 0';
        
        // Create hidden input to store combined value
        const hiddenInput = document.createElement('input');
        hiddenInput.type = 'hidden';
        hiddenInput.id = 'guess-input'; // Keep the original ID for compatibility
        hiddenInput.name = 'guess';
        
        // Create input boxes
        const inputRefs = [];
        for (let i = 0; i < wordLength; i++) {
            const inputBox = document.createElement('input');
            inputBox.type = 'text';
            inputBox.maxLength = 1;
            inputBox.className = 'otp-input';
            inputBox.setAttribute('data-index', i);
            
            // Set styles to ensure consistent display
            inputBox.style.width = '40px';
            inputBox.style.height = '40px';
            inputBox.style.minWidth = '40px';
            inputBox.style.maxWidth = '40px';
            inputBox.style.border = '1px solid #CED4DA';
            inputBox.style.textAlign = 'center';
            inputBox.style.fontSize = '1.5rem';
            inputBox.style.textTransform = 'uppercase';
            inputBox.style.padding = '0';
            inputBox.style.boxSizing = 'border-box';
            inputBox.style.flex = '0 0 auto';
            
            otpContainer.appendChild(inputBox);
            inputRefs.push(inputBox);
        }
        
        // Create guess container and verify button
        const guessContainer = document.getElementById('guess-container');
        if (!guessContainer) {
            // If no guess container exists, create one
            const newGuessContainer = document.createElement('div');
            newGuessContainer.id = 'guess-container';
            gameContainer.insertBefore(newGuessContainer, gameResults);
            guessContainer = newGuessContainer;
        }
        
        const verifyButton = document.createElement('button');
        verifyButton.textContent = 'Guess!';
        verifyButton.className = 'verify-button';
        verifyButton.id = 'guess-button'; // Keep the original ID for compatibility
        verifyButton.style.backgroundColor = '#28A745';
        verifyButton.style.color = 'white';
        verifyButton.style.padding = '12px 25px';
        verifyButton.style.width = '100%';
        verifyButton.style.border = 'none';
        verifyButton.style.fontSize = '1rem';
        verifyButton.style.fontWeight = '700';
        verifyButton.style.cursor = 'pointer';
        
        // Insert the OTP container in place of the word display
        wordDisplayElem.parentNode.insertBefore(otpContainer, wordDisplayElem);
        
        // Replace the original guess container content
        guessContainer.innerHTML = '';
        guessContainer.appendChild(hiddenInput);
        guessContainer.appendChild(verifyButton);
        
        // Add event listeners to input boxes
        inputRefs.forEach((input, index) => {
            // Handle input changes
            input.addEventListener('input', function(e) {
                const value = e.target.value;
                
                if (value) {
                    // Convert to uppercase
                    e.target.value = value.toUpperCase();
                    
                    // Move to next input if not the last one
                    if (index < wordLength - 1) {
                        inputRefs[index + 1].focus();
                    }
                }
                
                // Update hidden input with combined value
                updateHiddenInput();
            });
            
            // Handle keyboard navigation
            input.addEventListener('keydown', function(e) {
                // Submit on Enter key if all fields filled
                if (e.key === 'Enter') {
                    e.preventDefault();
                    const allFilled = inputRefs.every(inp => inp.value);
                    if (allFilled) {
                        verifyButton.click();
                    }
                }
                // Move to previous input on backspace if current is empty
                else if (e.key === 'Backspace' && !e.target.value && index > 0) {
                    inputRefs[index - 1].focus();
                }
                // Handle left/right arrow keys
                else if (e.key === 'ArrowLeft' && index > 0) {
                    inputRefs[index - 1].focus();
                }
                else if (e.key === 'ArrowRight' && index < wordLength - 1) {
                    inputRefs[index + 1].focus();
                }
            });
            
            // Handle paste event
            input.addEventListener('paste', function(e) {
                e.preventDefault();
                const pasteData = e.clipboardData.getData('text').replace(/\s/g, '').slice(0, wordLength);
                
                if (pasteData) {
                    // Fill as many boxes as we have characters
                    for (let i = 0; i < pasteData.length; i++) {
                        if (i < wordLength) {
                            inputRefs[i].value = pasteData[i].toUpperCase();
                        }
                    }
                    
                    // Update hidden input
                    updateHiddenInput();
                    
                    // Focus the next empty box or the last box
                    const nextEmptyIndex = inputRefs.findIndex(input => !input.value);
                    if (nextEmptyIndex !== -1 && nextEmptyIndex < wordLength) {
                        inputRefs[nextEmptyIndex].focus();
                    } else {
                        inputRefs[wordLength - 1].focus();
                    }
                }
            });
        });
        
        // Update hidden input function
        function updateHiddenInput() {
            hiddenInput.value = inputRefs.map(input => input.value || '').join('');
        }
        
        // Add click handler to the verify button
        verifyButton.addEventListener('click', function(event) {
            event.preventDefault();
            
            // Collect all values
            const guess = inputRefs.map(input => input.value || '').join('');
            
            // Validate all inputs are filled
            if (guess.length < wordLength) {
                if (gameResults) {
                    gameResults.textContent = 'Please fill all letters';
                }
                return;
            }
            
            // Update hidden input
            hiddenInput.value = guess;
            
            // Call our handleGuessSubmission function
            handleGuessSubmission(guess);
        });
        
        // Focus the first input box
        if (inputRefs.length > 0) {
            inputRefs[0].focus();
        }
    }

    // Initialize the OTP input
    setupOTPInput();
    const helpButton   = document.getElementById('help-button');
const helpModal    = document.getElementById('help-modal');
const closeModal   = helpModal?.querySelector('.modal-close');

function openHelp() {
  helpModal.setAttribute('open', '');
  helpModal.setAttribute('aria-hidden', 'false');
}
function closeHelp() {
  helpModal.removeAttribute('open');
  helpModal.setAttribute('aria-hidden', 'true');
}

helpButton?.addEventListener('click', openHelp);
closeModal ?.addEventListener('click', closeHelp);
// Close on backdrop click or Esc
helpModal?.addEventListener('click', e => { if (e.target === helpModal) closeHelp(); });
document.addEventListener('keydown', e => { if (e.key === 'Escape') closeHelp(); });
});