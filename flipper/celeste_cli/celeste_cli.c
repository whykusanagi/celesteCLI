#include <furi.h>
#include <furi_hal.h>
#include <gui/gui.h>
#include <input/input.h>
#include <storage/storage.h>
#include <furi_hal_usb.h>
#include <furi_hal_usb_hid.h>

#define MAX_MENU_ITEMS 10
#define MAX_COMMAND_LENGTH 256

typedef enum {
    AppStateSplash,
    AppStateMainMenu,
    AppStateTarotMenu,
    AppStateContentMenu,
    AppStateNSFWMenu,
    AppStateConfirm,
    AppStateExecuting,
    AppStateCustomInput
} AppState;

typedef enum {
    MenuCategoryMain,
    MenuCategoryTarot,
    MenuCategoryContent,
    MenuCategoryNSFW
} MenuCategory;

typedef struct {
    const char* name;
    const char* command;
    MenuCategory category;
} CelesteCommand;

// Command templates
static const CelesteCommand commands[] = {
    // Tarot commands
    {"3-Card Tarot", "celestecli --tarot\n", MenuCategoryTarot},
    {"Celtic Cross", "celestecli --tarot --spread celtic\n", MenuCategoryTarot},
    {"Divine Reading", "celestecli --divine\n", MenuCategoryTarot},
    {"Divine NSFW", "celestecli --divine-nsfw\n", MenuCategoryTarot},
    {"Tarot Parsed", "celestecli --tarot --parsed\n", MenuCategoryTarot},
    
    // Content generation - Twitter
    {"Twitter Short", "celestecli --format short --platform twitter --topic \"NIKKE\" --tone \"lewd\"\n", MenuCategoryContent},
    {"Twitter Teasing", "celestecli --format short --platform twitter --topic \"NIKKE\" --tone \"teasing\"\n", MenuCategoryContent},
    {"Twitter Chaotic", "celestecli --format short --platform twitter --topic \"NIKKE\" --tone \"chaotic\"\n", MenuCategoryContent},
    
    // Content generation - YouTube
    {"YouTube Desc", "celestecli --format long --platform youtube --topic \"Streaming\" --request \"include links to website, socials, products\"\n", MenuCategoryContent},
    
    // NSFW commands
    {"NSFW Text", "celestecli --nsfw --format short --platform twitter --topic \"NIKKE\" --tone \"explicit\"\n", MenuCategoryNSFW},
    {"NSFW Image", "celestecli --nsfw --image --request \"generate NSFW image of Celeste\"\n", MenuCategoryNSFW},
    {"List Models", "celestecli --nsfw --list-models\n", MenuCategoryNSFW},
    
    // End marker
    {NULL, NULL, MenuCategoryMain}
};

typedef struct {
    AppState state;
    uint8_t selected_item;
    uint8_t menu_start;
    MenuCategory current_category;
    const CelesteCommand* current_command;
    char custom_command[MAX_COMMAND_LENGTH];
    uint32_t splash_timer;
    bool running;
} CelesteApp;

static void render_splash(Canvas* canvas, CelesteApp* app) {
    UNUSED(app);
    canvas_clear(canvas);
    
    // Draw title
    canvas_set_font(canvas, FontPrimary);
    canvas_draw_str_aligned(canvas, 64, 10, AlignCenter, AlignTop, "CELESTE CLI");
    
    // Draw subtitle
    canvas_set_font(canvas, FontSecondary);
    canvas_draw_str_aligned(canvas, 64, 25, AlignCenter, AlignTop, "Remote Controller");
    
    // Draw Celeste symbol/icon placeholder
    canvas_draw_frame(canvas, 50, 35, 28, 20);
    canvas_draw_str_aligned(canvas, 64, 42, AlignCenter, AlignTop, "C");
    
    // Instructions
    canvas_set_font(canvas, FontSecondary);
    canvas_draw_str_aligned(canvas, 64, 58, AlignCenter, AlignBottom, "Press OK to start");
}

static uint8_t count_menu_items(MenuCategory category) {
    uint8_t count = 0;
    for(uint8_t i = 0; commands[i].name != NULL; i++) {
        if(commands[i].category == category) {
            count++;
        }
    }
    return count;
}

static const CelesteCommand* get_menu_item(MenuCategory category, uint8_t index) {
    uint8_t count = 0;
    for(uint8_t i = 0; commands[i].name != NULL; i++) {
        if(commands[i].category == category) {
            if(count == index) {
                return &commands[i];
            }
            count++;
        }
    }
    return NULL;
}

static void render_main_menu(Canvas* canvas, CelesteApp* app) {
    canvas_clear(canvas);
    
    // Header
    canvas_set_font(canvas, FontPrimary);
    canvas_draw_str(canvas, 2, 10, "CELESTE CLI");
    
    // Menu items
    const char* menu_items[] = {
        "Tarot Readings",
        "Content Gen",
        "NSFW Mode",
        "Settings"
    };
    
    uint8_t y_offset = 20;
    for(uint8_t i = 0; i < 4; i++) {
        if(i == app->selected_item) {
            canvas_draw_str(canvas, 4, y_offset, ">");
        }
        canvas_draw_str(canvas, 12, y_offset, menu_items[i]);
        y_offset += 10;
    }
    
    // Footer
    canvas_set_font(canvas, FontSecondary);
    canvas_draw_str_aligned(canvas, 64, 62, AlignCenter, AlignBottom, "OK=Select  Back=Exit");
}

static void render_submenu(Canvas* canvas, CelesteApp* app) {
    canvas_clear(canvas);
    
    // Header
    const char* headers[] = {
        [MenuCategoryTarot] = "TAROT READINGS",
        [MenuCategoryContent] = "CONTENT GEN",
        [MenuCategoryNSFW] = "NSFW MODE"
    };
    
    canvas_set_font(canvas, FontPrimary);
    canvas_draw_str(canvas, 2, 10, headers[app->current_category]);
    
    // Menu items
    uint8_t item_count = count_menu_items(app->current_category);
    uint8_t y_offset = 20;
    uint8_t visible_start = 0;
    uint8_t visible_count = 4;
    
    if(app->selected_item >= visible_count) {
        visible_start = app->selected_item - visible_count + 1;
    }
    
    for(uint8_t i = visible_start; i < item_count && (i - visible_start) < visible_count; i++) {
        const CelesteCommand* cmd = get_menu_item(app->current_category, i);
        if(cmd != NULL) {
            if(i == app->selected_item) {
                canvas_draw_str(canvas, 4, y_offset, ">");
            }
            canvas_draw_str(canvas, 12, y_offset, cmd->name);
            y_offset += 10;
        }
    }
    
    // Footer
    canvas_set_font(canvas, FontSecondary);
    canvas_draw_str_aligned(canvas, 64, 62, AlignCenter, AlignBottom, "OK=Send  Back=Menu");
}

static void render_confirm(Canvas* canvas, CelesteApp* app) {
    canvas_clear(canvas);
    
    canvas_set_font(canvas, FontPrimary);
    canvas_draw_str_aligned(canvas, 64, 10, AlignCenter, AlignTop, "SEND COMMAND?");
    
    if(app->current_command != NULL) {
        canvas_set_font(canvas, FontSecondary);
        
        // Show command name
        canvas_draw_str_aligned(canvas, 64, 25, AlignCenter, AlignTop, app->current_command->name);
        
        // Show command preview (truncated)
        char preview[32];
        strncpy(preview, app->current_command->command, 30);
        preview[30] = '\0';
        canvas_draw_str_aligned(canvas, 64, 40, AlignCenter, AlignTop, preview);
    }
    
    canvas_set_font(canvas, FontSecondary);
    canvas_draw_str_aligned(canvas, 64, 58, AlignCenter, AlignBottom, "OK=Send  Back=Cancel");
}

static void render_executing(Canvas* canvas, CelesteApp* app) {
    UNUSED(app);
    canvas_clear(canvas);
    
    canvas_set_font(canvas, FontPrimary);
    canvas_draw_str_aligned(canvas, 64, 30, AlignCenter, AlignCenter, "SENDING...");
    
    canvas_set_font(canvas, FontSecondary);
    canvas_draw_str_aligned(canvas, 64, 45, AlignCenter, AlignCenter, "Check host terminal");
}

static void render_callback(Canvas* canvas, void* ctx) {
    CelesteApp* app = (CelesteApp*)ctx;
    
    switch(app->state) {
    case AppStateSplash:
        render_splash(canvas, app);
        break;
    case AppStateMainMenu:
        render_main_menu(canvas, app);
        break;
    case AppStateTarotMenu:
    case AppStateContentMenu:
    case AppStateNSFWMenu:
        render_submenu(canvas, app);
        break;
    case AppStateConfirm:
        render_confirm(canvas, app);
        break;
    case AppStateExecuting:
        render_executing(canvas, app);
        break;
    default:
        break;
    }
}

static void send_key(HidKeyboardKey key, bool shift) {
    if(shift) {
        furi_hal_hid_kb_press(HID_KEYBOARD_LEFT_SHIFT);
    }
    furi_hal_hid_kb_press(key);
    furi_delay_ms(20);
    furi_hal_hid_kb_release(key);
    if(shift) {
        furi_hal_hid_kb_release(HID_KEYBOARD_LEFT_SHIFT);
    }
    furi_delay_ms(10);
}

static void send_char(char c) {
    if(c >= 'a' && c <= 'z') {
        send_key(HID_KEYBOARD_A + (c - 'a'), false);
    } else if(c >= 'A' && c <= 'Z') {
        send_key(HID_KEYBOARD_A + (c - 'A'), true);
    } else if(c == ' ') {
        send_key(HID_KEYBOARD_SPACEBAR, false);
    } else if(c == '\n') {
        send_key(HID_KEYBOARD_RETURN, false);
        furi_delay_ms(100);
    } else if(c == '-') {
        send_key(HID_KEYBOARD_MINUS, false);
    } else if(c == '"') {
        send_key(HID_KEYBOARD_APOSTROPHE, true);
    } else if(c == '0') {
        send_key(HID_KEYBOARD_0, false);
    } else if(c >= '1' && c <= '9') {
        send_key(HID_KEYBOARD_1 + (c - '1'), false);
    }
    // Add more special characters as needed
}

static void send_command(const char* command) {
    // Enable USB HID
    furi_hal_usb_set_config(&usb_hid, NULL);
    
    // Wait for USB connection (with timeout)
    uint32_t timeout = 0;
    while(!furi_hal_usb_is_connected() && timeout < 500) {
        furi_delay_ms(10);
        timeout++;
    }
    
    if(!furi_hal_usb_is_connected()) {
        return; // USB not connected
    }
    
    // Small delay to ensure host is ready
    furi_delay_ms(500);
    
    // Type the command
    for(size_t i = 0; i < strlen(command); i++) {
        send_char(command[i]);
    }
    
    // Disable USB HID
    furi_hal_usb_set_config(&usb_hid, NULL);
}

static void input_callback(InputEvent* input_event, void* ctx) {
    furi_assert(ctx);
    CelesteApp* app = (CelesteApp*)ctx;
    
    if(input_event->type == InputTypePress) {
        switch(input_event->key) {
        case InputKeyUp:
            if(app->state == AppStateMainMenu) {
                if(app->selected_item > 0) {
                    app->selected_item--;
                }
            } else if(app->state == AppStateTarotMenu || 
                     app->state == AppStateContentMenu || 
                     app->state == AppStateNSFWMenu) {
                uint8_t item_count = count_menu_items(app->current_category);
                if(app->selected_item > 0) {
                    app->selected_item--;
                }
            }
            break;
            
        case InputKeyDown:
            if(app->state == AppStateMainMenu) {
                if(app->selected_item < 3) {
                    app->selected_item++;
                }
            } else if(app->state == AppStateTarotMenu || 
                     app->state == AppStateContentMenu || 
                     app->state == AppStateNSFWMenu) {
                uint8_t item_count = count_menu_items(app->current_category);
                if(app->selected_item < item_count - 1) {
                    app->selected_item++;
                }
            }
            break;
            
        case InputKeyOk:
            if(app->state == AppStateSplash) {
                app->state = AppStateMainMenu;
                app->selected_item = 0;
            } else if(app->state == AppStateMainMenu) {
                switch(app->selected_item) {
                case 0: // Tarot
                    app->state = AppStateTarotMenu;
                    app->current_category = MenuCategoryTarot;
                    app->selected_item = 0;
                    break;
                case 1: // Content
                    app->state = AppStateContentMenu;
                    app->current_category = MenuCategoryContent;
                    app->selected_item = 0;
                    break;
                case 2: // NSFW
                    app->state = AppStateNSFWMenu;
                    app->current_category = MenuCategoryNSFW;
                    app->selected_item = 0;
                    break;
                case 3: // Settings (placeholder)
                    // TODO: Implement settings
                    break;
                }
            } else if(app->state == AppStateTarotMenu || 
                      app->state == AppStateContentMenu || 
                      app->state == AppStateNSFWMenu) {
                const CelesteCommand* cmd = get_menu_item(app->current_category, app->selected_item);
                if(cmd != NULL) {
                    app->current_command = cmd;
                    app->state = AppStateConfirm;
                }
            } else if(app->state == AppStateConfirm) {
                if(app->current_command != NULL) {
                    app->state = AppStateExecuting;
                    send_command(app->current_command->command);
                    furi_delay_ms(1000);
                    app->state = AppStateMainMenu;
                    app->selected_item = 0;
                }
            }
            break;
            
        case InputKeyBack:
            if(app->state == AppStateSplash) {
                // Exit app
                app->running = false;
                break;
            } else if(app->state == AppStateMainMenu) {
                // Exit app
                app->running = false;
                break;
            } else if(app->state == AppStateTarotMenu || 
                     app->state == AppStateContentMenu || 
                     app->state == AppStateNSFWMenu) {
                app->state = AppStateMainMenu;
                app->selected_item = 0;
            } else if(app->state == AppStateConfirm) {
                app->state = AppStateTarotMenu;
                if(app->current_category == MenuCategoryContent) {
                    app->state = AppStateContentMenu;
                } else if(app->current_category == MenuCategoryNSFW) {
                    app->state = AppStateNSFWMenu;
                }
            } else if(app->state == AppStateExecuting) {
                app->state = AppStateMainMenu;
                app->selected_item = 0;
            }
            break;
            
        default:
            break;
        }
    }
}

int32_t celeste_cli_app(void* p) {
    UNUSED(p);
    
    CelesteApp* app = malloc(sizeof(CelesteApp));
    app->state = AppStateSplash;
    app->selected_item = 0;
    app->current_category = MenuCategoryMain;
    app->current_command = NULL;
    app->splash_timer = 0;
    app->running = true;
    memset(app->custom_command, 0, MAX_COMMAND_LENGTH);
    
    ViewPort* view_port = view_port_alloc();
    view_port_draw_callback_set(view_port, render_callback, app);
    view_port_input_callback_set(view_port, input_callback, app);
    
    Gui* gui = furi_record_open(RECORD_GUI);
    gui_add_view_port(gui, view_port, GuiLayerFullscreen);
    
    // Main loop
    while(app->running) {
        if(app->state == AppStateSplash) {
            app->splash_timer++;
            if(app->splash_timer >= 300) {
                // Auto-advance from splash after 15 seconds (300 * 50ms)
                app->state = AppStateMainMenu;
                app->selected_item = 0;
            }
        }
        view_port_update(view_port);
        furi_delay_ms(50);
    }
    
    // Cleanup
    gui_remove_view_port(gui, view_port);
    view_port_free(view_port);
    furi_record_close(RECORD_GUI);
    free(app);
    
    return 0;
}

