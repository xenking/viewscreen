$('.tracker .menu .item').click(function () {
    $('.tracker.menu .active').removeClass('active');
    let tab = $(this).data('tab');
    $(".ui.tab[data-tab]")
        .filter(function () {
            return $(this).data("tab") === tab;
        }).addClass('active');
    $(this).addClass('active');
    $('#tab').val(tab);
    $('#page').val(1);
    $('button.search').submit();
});

$(window).bind('resultsComplete', function () {
    console.log('resultsComplete');
    // set tablesort.js
    $('table').tablesort();
    // set S\L icon
    $('table .circle.icon').popup({
        hoverable: true,
        inline: true,
        delay: {
            show: 100,
            hide: 500
        },
        position: 'right center',
        lastResort: 'right center'
    });
    // set DL button handler
    setModals();
    // set pagination slider
    onResize(true);
    setSliders();
});

function setModals() {
    $('table button.download').click(function () {
        let saveForm = $('.ui.form.save');
        saveForm.first().append(
            $('#target').val($(this).data("value"))
        );
        saveForm.find("#name").val($(this).parent().parent().find('.title').text().trim());
        saveForm.attr('action', $(this).data("action"));
        $('.ui.modal').modal('show');
        $(document).on('click', '.actions>.positive.button', {
            data: this
        }, function (e) {
            let data = e.data.data;
            if ($('.ui.form.save').form('is valid') && !$(data).hasClass('downloading')) {
                let target = $(data).siblings(".downloading");
                $(data).toggle();
                $(target).toggle();
                $(data).addClass('downloading');
            }
        });
    });

    $(".ui.modal").modal({
        closable: false,
        onApprove: function () {
            let saveForm = $('.ui.form.save');
            saveForm.submit();
            if (!saveForm.form('is valid')) {
                return false;
            }
            return true;
        }
    });
    $('.ui.checkbox').checkbox();
    $('.ui.checkbox.subs').checkbox({
        onChange: function () {
            $('.field.subs').toggle()
        }
    });

    setSaveForm();
}

function setSliders() {
    $(".ui.results.menu").each(function () {
        let menu = this;

        $(".scroll", this).each(function () {
            let el = this;
            $(this).slider(() => buttonScroll(el, menu), 250)
        })
    })
    // set pagination slider submit button
    $('.ui.results.menu').children('.item:not(.scroll)').each(function () {
        $(this).click(function () {
            $('#page').val($(this).text());
            $('.ui.pagination.menu .active').removeClass('active');
            $(this).addClass('active');
            $('form.search').submit();
        })
    });
}

$(document).on('click', '.ui.history.menu a.item', function (e) {
    e.preventDefault();
    let menu = $(".ui.history.menu");
    $('.active', menu).removeClass('active');
    let history = $(this).data('history');
    $(this).addClass('active');
    $('#query').val(history);
    $('#page').val(1);
    $('form.search').submit();
});

function addHistory(query) {
    let history = $(".ui.history.menu a.item").map(function () {
        return $(this).text();
    });
    if ($.inArray('"' + query + '"', history) === -1) {
        $(".ui.history.menu").append('<a href="#" class="active item" data-history="' + query + '">&quot;' + query + '&quot;</a>');
    }

}