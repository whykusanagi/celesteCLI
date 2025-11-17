#!/usr/bin/env python3
"""
Fix placeholder reversed meanings in tarot_cards.json
"""
import json
import sys

def get_reversed_meaning(card_name, suit=None):
    """Generate appropriate reversed meaning based on card name and suit"""
    name_lower = card_name.lower()
    
    # Suit-based reversed meanings
    if "cups" in name_lower:
        return "Blocked emotions, emotional imbalance, or unexpressed feelings."
    elif "wands" in name_lower:
        return "Lack of direction, creative blocks, or delayed action."
    elif "swords" in name_lower:
        return "Mental confusion, communication breakdown, or inner conflict."
    elif "pentacles" in name_lower or "coins" in name_lower:
        return "Financial instability, material loss, or lack of grounding."
    elif "the fool" in name_lower:
        return "Recklessness, poor judgment, or taking unnecessary risks."
    elif "the magician" in name_lower:
        return "Manipulation, untapped talents, or lack of focus."
    elif "the high priestess" in name_lower:
        return "Repressed intuition, disconnect from inner voice, or secrets."
    elif "the empress" in name_lower:
        return "Creative blocks, dependence on others, or lack of growth."
    elif "the emperor" in name_lower:
        return "Tyranny, rigidity, or abuse of power."
    elif "the hierophant" in name_lower:
        return "Rebellion, unconventional beliefs, or challenging tradition."
    elif "the lovers" in name_lower:
        return "Disharmony, imbalance, or poor choices in relationships."
    elif "the chariot" in name_lower:
        return "Lack of control, directionless, or inner conflict."
    elif "strength" in name_lower:
        return "Weakness, self-doubt, or inner strength blocked."
    elif "the hermit" in name_lower:
        return "Isolation, withdrawal, or refusal to seek guidance."
    elif "wheel of fortune" in name_lower:
        return "Bad luck, resistance to change, or cycles of negativity."
    elif "justice" in name_lower:
        return "Injustice, dishonesty, or lack of accountability."
    elif "the hanged man" in name_lower:
        return "Stagnation, delay, or resistance to necessary sacrifice."
    elif "death" in name_lower:
        return "Resistance to change, stagnation, or fear of transformation."
    elif "temperance" in name_lower:
        return "Imbalance, excess, or lack of moderation."
    elif "the devil" in name_lower:
        return "Breaking free from bondage, releasing limiting beliefs."
    elif "the tower" in name_lower:
        return "Avoiding necessary change, internal collapse, or false security."
    elif "the star" in name_lower:
        return "Despair, loss of faith, or blocked hope."
    elif "the moon" in name_lower:
        return "Clarity emerging, releasing illusions, or facing fears."
    elif "the sun" in name_lower:
        return "Temporary darkness, overconfidence, or blocked joy."
    elif "judgment" in name_lower:
        return "Self-doubt, lack of self-awareness, or refusal to reflect."
    elif "the world" in name_lower:
        return "Incompletion, lack of closure, or feeling stuck."
    
    # Number-based meanings for minor arcana
    if "ace" in name_lower:
        return "Blocked potential, missed opportunities, or lack of new beginnings."
    elif "two" in name_lower:
        return "Imbalance, conflict, or lack of partnership."
    elif "three" in name_lower:
        return "Lack of collaboration, isolation, or creative blocks."
    elif "four" in name_lower:
        return "Restlessness, instability, or lack of security."
    elif "five" in name_lower:
        return "Recovery, moving past conflict, or finding resolution."
    elif "six" in name_lower:
        return "Stuck in past, inability to move forward, or imbalance."
    elif "seven" in name_lower:
        return "Lack of planning, giving up too easily, or self-doubt."
    elif "eight" in name_lower:
        return "Lack of progress, giving up, or moving too slowly."
    elif "nine" in name_lower:
        return "Lack of completion, giving up, or inability to finish."
    elif "ten" in name_lower:
        return "Burden, responsibility, or completion blocked."
    elif "page" in name_lower:
        return "Lack of curiosity, immaturity, or poor communication."
    elif "knight" in name_lower:
        return "Lack of action, impulsiveness, or recklessness."
    elif "queen" in name_lower:
        return "Insecurity, dependence, or lack of inner strength."
    elif "king" in name_lower:
        return "Tyranny, abuse of power, or lack of leadership."
    
    # Generic fallback
    return "The reversed position suggests blocked energy, internal resistance, or the need for reflection on this card's themes."

def fix_tarot_json(input_file, output_file):
    """Fix all placeholder reversed meanings in the tarot JSON file"""
    with open(input_file, 'r', encoding='utf-8') as f:
        cards = json.load(f)
    
    fixed_count = 0
    for card in cards:
        if 'reversed' in card and isinstance(card['reversed'], str):
            if card['reversed'].startswith('Reversed meaning of '):
                suit = card.get('suit', '')
                card['reversed'] = get_reversed_meaning(card['name'], suit)
                fixed_count += 1
    
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(cards, f, indent=2, ensure_ascii=False)
    
    print(f"Fixed {fixed_count} cards with placeholder reversed meanings")
    print(f"Updated JSON saved to {output_file}")
    return fixed_count

if __name__ == "__main__":
    input_file = "tarot_cards.json"
    output_file = "tarot_cards_fixed.json"
    
    if len(sys.argv) > 1:
        input_file = sys.argv[1]
    if len(sys.argv) > 2:
        output_file = sys.argv[2]
    
    fix_tarot_json(input_file, output_file)

