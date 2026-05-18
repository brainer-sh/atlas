// A simple shape class.
class Shape {
public:
    virtual float area() const = 0;
};

float computeArea(Shape *s) {
    return s->area();
}

template<typename T>
class Stack {
public:
    void push(T val);
    T pop();
};

#define MAX_STACK_SIZE 256
