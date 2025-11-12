import json
import random
import requests

TAROT_JSON_URL = "https://s3.whykusanagi.xyz/tarot_cards.json"

THREE_CARD_POSITIONS = [
    "Past",
    "Present",
    "Future"
]

CELTIC_CROSS_POSITIONS = [
    "Present Situation",
    "Challenge/Crossing",
    "Distant Past",
    "Recent Past",
    "Possible Future",
    "Near Future",
    "Your Approach",
    "External Influences",
    "Hopes and Fears",
    "Final Outcome"
]

def main(args):
    """
    DigitalOcean Function to generate a Tarot reading.
    Supports both Three Card Spread and Celtic Cross Spread.
    Safe for web UI, curl, or LLM function call.
    """
    args = args or {}
    method = args.get("__ow_method", "get").lower()
    
    # Get spread type from request body (for POST) or args (for GET)
    spread_type = "three"  # default
    if method == "post":
        # For POST requests, check the request body
        body = args.get("body", {})
        if isinstance(body, str):
            try:
                body = json.loads(body)
            except:
                body = {}
        spread_type = body.get("spread_type", "three")
    else:
        # For GET requests, check query params
        spread_type = args.get("spread_type", "three")
    
    # Normalize spread type
    spread_type = spread_type.lower()
    if spread_type not in ["three", "celtic"]:
        spread_type = "three"  # default to three if invalid

    if method == "get":
        print(f"GET request received for {spread_type} spread", flush=True)

    try:
        response = requests.get(TAROT_JSON_URL)
        response.raise_for_status()
        deck = response.json()

        # Determine number of cards and positions based on spread type
        if spread_type == "celtic":
            draw_count = 10
            positions = CELTIC_CROSS_POSITIONS
            spread_name = "Celtic Cross Spread"
        else:  # three
            draw_count = 3
            positions = THREE_CARD_POSITIONS
            spread_name = "Three Card Spread"

        # Validate we have enough cards in the deck
        if len(deck) < draw_count:
            return {"error": f"Not enough cards in deck. Need {draw_count}, but only {len(deck)} available."}

        # Draw cards without replacement - random.sample ensures no duplicates
        # Each card can only be drawn once per spread
        drawn = random.sample(deck, draw_count)

        # Build spread
        spread = []
        for position, card in zip(positions, drawn):
            # Randomly determine orientation
            orientation = random.choice(["upright", "reversed"])
            
            # Get the meaning based on orientation from the card JSON
            if orientation == "reversed" and "reversed" in card:
                meaning = card["reversed"]
            elif orientation == "upright" and "upright" in card:
                meaning = card["upright"]
            else:
                # Fallback if orientation key is missing
                meaning = card.get(orientation, "Meaning not available.")
            
            spread.append({
                "position": position,
                "card_name": card["name"],
                "orientation": orientation,
                "card_meaning": meaning
            })

        return {
            "spread_name": spread_name,
            "spread_type": spread_type,
            "cards": spread
        }

    except Exception as e:
        return {"error": f"Failed to generate reading: {str(e)}"}

