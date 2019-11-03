function saveImageAs(canvas, filename, type, quality) {
    let anchorElement, event, blob;
    type = type ? type : "png";
    anchorElement = document.createElement('a');
    if (type.toLowerCase() === "jpg" || type.toLowerCase() === "jpeg") {
        quality = quality ? quality : 0.9;
        anchorElement.href = canvas.toDataURL("image/jpeg", quality);
    } else {
        anchorElement.href = canvas.toDataURL();
    }

    if (anchorElement.download !== undefined) {
        anchorElement.download = filename + "." + type;
        if (typeof MouseEvent === "function") {
            event = new MouseEvent("click", {
                view: window,
                bubbles: true,
                cancelable: true
            });
            anchorElement.dispatchEvent(event);
        } else if (anchorElement.fireEvent) {
            anchorElement.fireEvent("onclick");
        }
    }
}

let Button = videojs.getComponent('Button');
let PrevButton = videojs.extend(Button, {
    constructor: function () {
        Button.apply(this, arguments);
        this.addClass('icon-chevron-left');
        this.controlText("Previous");
    },
    handleClick: function () {
        console.log('click');
        player.playlist.previous();
    }
});

let NextButton = videojs.extend(Button, {
    constructor: function () {
        Button.apply(this, arguments);
        this.addClass('icon-chevron-right');
        this.controlText("Next");
    },

    handleClick: function () {
        console.log('click');
        player.playlist.next();
    }
});

let ScreenshotButton = videojs.extend(Button, {
    constructor: function () {
        Button.apply(this, arguments);
        this.addClass('icon-image');
        this.controlText("Save screen");
    },

    handleClick: function () {
        let video = document.querySelector('video');
        let canvas = document.createElement('canvas');
        let ratio = video.videoWidth / video.videoHeight;
        let w = video.videoWidth;
        let h = parseInt(w / ratio, 10);
        canvas.width = w;
        canvas.height = h;
        canvas.getContext('2d').fillRect(0, 0, w, h);
        canvas.getContext('2d').drawImage(video, 0, 0, w, h);
        saveImageAs(canvas, "snap");
    }
});

videojs.registerComponent('nextButton', NextButton);
videojs.registerComponent('prevButton', PrevButton);
videojs.registerComponent('screenshotButton', ScreenshotButton);

const ControlBar = videojs.getComponent("ControlBar");
ControlBar.prototype.options_ = {
    loadEvent: 'play',
    children: ['prevButton', 'playToggle', 'nextButton', 'volumePanel', 'progressControl', 'liveDisplay', 'remainingTimeDisplay', 'durationDisplay', 'customControlSpacer', 'playbackRateMenuButton', 'chaptersButton', 'descriptionsButton', 'subsCapsButton', 'audioTrackButton', 'screenshotButton', 'fullscreenToggle']
};

let player = videojs(document.querySelector('.video-player'), {
    controls: true,
    fluid: true,
    preload: 'auto',
    plugins: {
        controlspreview: {
            loadOnStart: true,
        }
    }
});

player.on('playlistitem', function(event, video) {
    player.dock({
        title: video.title,
        description: video.description
    });
});

player.ready(function () {
    let controlBar = player.getChild('controlBar');
    let remainingTime = controlBar.getChild('remainingTimeDisplay');
    let durationTime = controlBar.getChild('durationDisplay');
    durationTime.addClass('vjs-hidden');
    remainingTime.on('click', function(){
        console.log("hide remaining time");
        this.hide();
        durationTime.show();
    });

    durationTime.on('click', function(){
        console.log("hide duration time");
        this.hide();
        remainingTime.show();
    });
    this.hotkeys({
        volumeStep: 0.1,
        seekStep: function(e) {
            if (e.altKey && e.shiftKey) {
                return 60;
            } else if (e.shiftKey) {
                return 30;
            } else if (e.altKey) {
                return 10;
            } else {
                return 5;
            }
        },
        forwardKey: function(e) {
            if (e.ctrlKey){
                return false;
            } else {
                return (e.which === 39 || e.which === 176);
            }
        },
        rewindKey: function(e) {
            if (e.ctrlKey){
                return false;
            } else {
                return (e.which === 37 || e.which === 177);
            }
        },
        enableModifiersForNumbers: false,
        customKeys: {
            NextEpisodeKey: {
                key: function(e) {
                    return (e.ctrlKey && e.which === 39);
                },
                handler: function() {
                    NextButton.prototype.handleClick();
                }
            },
            PrevEpisodeKey: {
                key: function(e) {
                    return (e.ctrlKey && e.which === 37);
                },
                handler: function() {
                    PrevButton.prototype.handleClick();
                }
            },
        }
    });
});

