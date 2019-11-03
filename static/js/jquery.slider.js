(function ($) {

    class ScrollObj {
        nextTime = 0;
        state = 0;
        mousedownFired = false;
        delay = 500;

        constructor(target, func, delay) {
            this.target = target;
            this.delay = delay;
            this.func = func;
            this.handlerMouseDown = this.handlerMouseDown.bind(this);
            this.handlerMouseUp = this.handlerMouseUp.bind(this);
            this.watcher = this.watcher.bind(this);
        }

        watcher(time) {
            if (this.mousedownFired) {
                requestAnimationFrame((time) => this.watcher(time));
            }

            if (time < this.nextTime) {
                return;
            }
            this.nextTime = time + this.delay;
            if (this.state !== 0) {

                this.func();
            }

        }

        handlerMouseDown(e) {
            e.preventDefault();
            e.stopPropagation();
            this.state = 1;
            this.mousedownFired = true;
            requestAnimationFrame((time) => this.watcher(time));
        }

        handlerMouseUp(e) {
            e.preventDefault();
            e.stopPropagation();
            if (this.mousedownFired) {
                this.mousedownFired = false;
                this.state = 0;
            }
        }

        listen() {
            this.target.mousedown(this.handlerMouseDown);
            this.target.mouseup(this.handlerMouseUp);
            this.target.mouseout(this.handlerMouseUp);
        }
    }

    $.fn.slider = function (func, delay) {
        let elem = new ScrollObj(this, func, delay);
        elem.listen();
    }
})(jQuery);