; Direct function: void foo(...)
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @definition.function

; Pointer-return function: SDL_GPUDevice *SDL_GPUCreateDevice(...)
(function_definition
  declarator: (pointer_declarator
    declarator: (function_declarator
      declarator: (identifier) @name))) @definition.function

; Typedef struct
(type_definition
  type: (struct_specifier
    name: (type_identifier) @name)) @definition.struct

; Typedef enum
(type_definition
  type: (enum_specifier
    name: (type_identifier) @name)) @definition.enum

; Object-like macro: #define FOO 42
(preproc_def
  name: (identifier) @name) @definition.macro

; Function-like macro: #define FOO(x) (x+1)
(preproc_function_def
  name: (identifier) @name) @definition.macro
