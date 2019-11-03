window.pollerJob = {};
window.poller = function (target, url, delay) {
    // Don't allow duplicate targets, so it's safe to call poller multiple times.
    if (window.pollerJob[target] === "active") {
        return;
    }
    window.pollerJob[target] = "active";

    let old = '';
    let p = function () {
        // Target is gone, so we're done.
        if ($(target).length === 0) {
            delete window.pollerJob[target];
            return
        }

        // Make the request.
        $.ajax({
            url: url,
            type: 'GET',
            success: function (data) {
                // A doctype tag indicates we're getting a full HTML response, not a fragment.
                if (data.substring(0, 50).toLowerCase().indexOf("doctype") !== -1) {
                    return;
                }
                // We only update the target if there is a change.
                if (data !== old) {
                    $(target).html(data);
                    old = data;
                }
            },
            complete: function () {
                setTimeout(p, delay);
            }
        });
    };
    p();
};

$(document).ready(function () {
    // show dropdown on hover
    $('.main.menu .ui.dropdown').dropdown({
        on: 'hover'
    });
    onResize(true);
})

function onResize(scroll = false) {
    $(".ui.pagination.menu").each(function() {
        let menu = this;
        $(this).scrollTo($('.active.item', this).text() >= 3 ? $('.active.item', this).prev().prev() : $('.active.item', this).prev());
        buttonScroll($(".scroll.left", menu), menu, !scroll);
        buttonScroll($(".scroll.right", menu), menu, !scroll);

    })
}

$(window).resize(function() {
    if (this.resizeTO) clearTimeout(this.resizeTO);
    this.resizeTO = setTimeout(function() {
        $(this).trigger('resizeEnd');
    }, 250);
});

$(window).bind('resizeEnd', onResize);

$(document).ready(function() {
    return onResize(true);
});

$(".ui.pagination.menu").each(function() {
    let menu = this;

    $(".scroll", this).each(function() {
        let el = this;
        $(this).slider(() => buttonScroll(el, menu), 175)
    })
})

function buttonScroll(el, menu, scroll = true) {
    let offsetLeft = 25;
    let offsetRight = 45;
    if ($(el).hasClass('scroll right')) {
        if (scroll) {
            $(menu).scrollTo("+=50px");
        }
        if (menu.scrollLeft >= (menu.scrollWidth - menu.clientWidth - offsetRight)) {
            $(el).hide();
        } else {
            if (scroll) {
                $(el).siblings('.scroll').show();
            }
            $(el).show();
        }
    }
    if ($(el).hasClass('scroll left')) {
        if (scroll) {
            $(menu).scrollTo("-=50px");
        }
        if (menu.scrollLeft <= offsetLeft) {
            $(el).hide();
        } else {
            if (scroll) {
                $(el).siblings('.scroll').show();
            }
            $(el).show();
        }
    }
}

