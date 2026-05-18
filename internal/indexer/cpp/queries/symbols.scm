; Free function: void foo(...)
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @name)) @definition.function

; Pointer-return function: Foo *bar(...)
(function_definition
  declarator: (pointer_declarator
    declarator: (function_declarator
      declarator: (identifier) @name))) @definition.function

; Out-of-line method: void Foo::bar(...)
(function_definition
  declarator: (function_declarator
    declarator: (qualified_identifier) @name)) @definition.method

; Pointer-return out-of-line method: Foo *Bar::baz(...)
(function_definition
  declarator: (pointer_declarator
    declarator: (function_declarator
      declarator: (qualified_identifier) @name))) @definition.method

; Class definition
(class_specifier
  name: (type_identifier) @name) @definition.class

; Struct definition (C++ style, no typedef)
(struct_specifier
  name: (type_identifier) @name) @definition.struct

; Enum definition (scoped or unscoped)
(enum_specifier
  name: (type_identifier) @name) @definition.enum

; Typedef struct
(type_definition
  type: (struct_specifier
    name: (type_identifier) @name)) @definition.struct

; Typedef enum
(type_definition
  type: (enum_specifier
    name: (type_identifier) @name)) @definition.enum

; Template function
(template_declaration
  (function_definition
    declarator: (function_declarator
      declarator: (identifier) @name))) @definition.template

; Template pointer-return function
(template_declaration
  (function_definition
    declarator: (pointer_declarator
      declarator: (function_declarator
        declarator: (identifier) @name)))) @definition.template

; Template class
(template_declaration
  (class_specifier
    name: (type_identifier) @name)) @definition.template

; Object-like macro: #define FOO 42
(preproc_def
  name: (identifier) @name) @definition.macro

; Function-like macro: #define FOO(x) (x+1)
(preproc_function_def
  name: (identifier) @name) @definition.macro
