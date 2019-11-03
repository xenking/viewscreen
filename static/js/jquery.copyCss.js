(function ($) {

    $.fn.getStyles = function (only, except) {

        var product = {};

        var style;

        var name;

        if (only && only instanceof Array) {

            for (var i = 0, l = only.length; i < l; i++) {
                name = only[i];
                product[name] = this.css(name);
            }

        } else {

            if (this.length) {

                var dom = this.get(0);

                if (window.getComputedStyle) {

                    var pattern = /\-([a-z])/g;
                    var uc = function (a, b) {
                        return b.toUpperCase();
                    };
                    var camelize = function (string) {
                        return string.replace(pattern, uc);
                    };

                    if (style = window.getComputedStyle(dom, null)) {
                        var camel, value;

                        if (style.length) {
                            for (var i = 0, l = style.length; i < l; i++) {
                                name = style[i];
                                camel = camelize(name);
                                value = style.getPropertyValue(name);
                                product[camel] = value;
                            }
                        } else {
                            for (name in style) {
                                camel = camelize(name);
                                value = style.getPropertyValue(name) || style[name];
                                product[camel] = value;
                            }
                        }
                    }
                } else if (style = dom.currentStyle) {
                    for (name in style) {
                        product[name] = style[name];
                    }
                } else if (style = dom.style) {
                    for (name in style) {
                        if (typeof style[name] != 'function') {
                            product[name] = style[name];
                        }
                    }
                }
            }
        }

        if (except && except instanceof Array) {
            for (var i = 0, l = except.length; i < l; i++) {
                name = except[i];
                delete product[name];
            }
        }

        return product;

    };

    $.fn.copyCSS = function (source, only, except) {
        var styles = $(source).getStyles(only, except);
        this.css(styles);

        return this;
    };

})(jQuery);