import re

with open('scripts/bots/populate.py', 'r', encoding='utf-8') as f:
    text = f.read()

start_idx = text.find('TOPIC_TEMPLATES = [')
end_idx = text.find('NAMES_AND_BIOS = [')

new_code = '''TOPIC_TEMPLATES = [
    {
        "category": "splitter_evangelist",
        "hashtags": ["#Splitter", "#Federation", "#Web3", "#FutureOfSocial"],
        "prompt": "Write a short casual social media post (1-3 sentences) praising Splitter, the new federated app we are on. Mention decentralization or no algorithms. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "splitter_support",
        "hashtags": ["#SplitterApp", "#TechSupport", "#Federation"],
        "prompt": "Write a short casual social media post (1-2 sentences) about how smooth the Splitter UI is, or seeing live trending hashtags here. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "crypto_decentralization",
        "hashtags": ["#Crypto", "#Blockchain", "#DeFi", "#Decentralization"],
        "prompt": "Write a short casual social media post (1-3 sentences) about decentralization and taking back control of data. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_tech",
        "hashtags": ["#Programming", "#AI", "#Coding", "#Developer"],
        "prompt": "Write a short casual social media post (1-3 sentences) about coding, learning a new technology, or building software. Don't mention specific apps. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_life_1",
        "hashtags": ["#WeekendVibes", "#Coffee", "#DailyGrind"],
        "prompt": "Write a short casual social media post (1-3 sentences) about enjoying a coffee, morning routines, or weekend plans. Sound like a regular person posting an update. Include 1 hashtag like {hashtags}."
    },
    {
        "category": "generic_life_2",
        "hashtags": ["#Foodie", "#Cooking", "#LunchBreak"],
        "prompt": "Write a short casual social media post (1-2 sentences) about trying a new recipe, eating good food, or a lunch break. Sound like a regular person. Include 1 hashtag like {hashtags}."
    },
    {
        "category": "generic_entertainment",
        "hashtags": ["#Gaming", "#Movies", "#NowWatching", "#Music"],
        "prompt": "Write a short casual social media post (1-3 sentences) about watching a good movie, listening to music, or playing a video game. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_thoughts",
        "hashtags": ["#RandomThoughts", "#Mood", "#JustThinking"],
        "prompt": "Write a short casual social media post (1-2 sentences) sharing a random shower thought, minor observation about life, or a sudden realization. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_nature",
        "hashtags": ["#Nature", "#Outdoors", "#Walking", "#Sunlight"],
        "prompt": "Write a short casual social media post (1-2 sentences) about going for a walk, enjoying the weather, or spending time outdoors. Include 1-2 hashtags like {hashtags}."
    }
]

'''

new_text = text[:start_idx] + new_code + text[end_idx:]

with open('scripts/bots/populate.py', 'w', encoding='utf-8') as f:
    f.write(new_text)

print("Successfully replaced templates to have 50-60% normal conversations!")
