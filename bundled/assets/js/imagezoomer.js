if ( typeof Object.create !== 'function' ) {
    Object.create = function( obj ) {
        function F() {};
        F.prototype = obj;
        return new F();
    }; //func
} // if

(function( $, window, document, undefined ) {
    var ImageZoom = {
        init: function( options, elem ) {
            var self = this;
            self.elem = elem;
            self.$elem = $( elem );
            self.imageSrc = self.$elem.data("zoom-image") ? self.$elem.data("zoom-image") : self.$elem.attr("src");
            self.options = $.extend( {}, $.fn.imageZoom.options, options );
            if(self.options.tint) {
                self.options.lensColour = "none", //colour of the lens background
                    self.options.lensOpacity =  "1" //opacity of the lens
            } // if
            if(self.options.zoomType == "inner") {
                self.options.showLens = false;
            } // if
            self.$elem.parent().removeAttr('title').removeAttr('alt');
            self.zoomImage = self.imageSrc;
            self.refresh( 1 );
            $('#'+self.options.gallery + ' a').click( function(e) {
                if(self.options.galleryActiveClass){
                    $('#'+self.options.gallery + ' a').removeClass(self.options.galleryActiveClass);
                    $(this).addClass(self.options.galleryActiveClass);
                } // if
                e.preventDefault();
                if($(this).data("zoom-image")){self.zoomImagePre = $(this).data("zoom-image")}
                else{self.zoomImagePre = $(this).data("image");}
                self.swaptheimage($(this).data("image"), self.zoomImagePre);
                return false;
            }); // func
        }, // func
        refresh: function( length ) {
            var self = this;
            setTimeout(function() {
                self.fetch(self.imageSrc);
            }, length || self.options.refresh );
        }, // func
        fetch: function(imgsrc) {
            var self = this;
            var newImg = new Image();
            newImg.onload = function() {
                self.largeWidth = newImg.width;
                self.largeHeight = newImg.height;
                self.startZoom();
                self.currentImage = self.imageSrc;
                self.options.onZoomedImageLoaded(self.$elem);
            } // func
            newImg.src = imgsrc;
            return;
        }, // func
        startZoom: function( ) {
            var self = this;
            self.nzWidth = self.$elem.width();
            self.nzHeight = self.$elem.height();
            self.isWindowActive = false;
            self.isLensActive = false;
            self.isTintActive = false;
            self.overWindow = false;
            if(self.options.imageCrossfade){
                self.zoomWrap = self.$elem.wrap('<div style="height:'+self.nzHeight+'px;width:'+self.nzWidth+'px;" class="zoomWrapper" />');
                self.$elem.css('position', 'absolute');
            } // if
            self.zoomLock = 1;
            self.scrollingLock = false;
            self.changeBgSize = false;
            self.currentZoomLevel = self.options.zoomLevel;
            self.nzOffset = self.$elem.offset();
            self.widthRatio = (self.largeWidth/self.currentZoomLevel) / self.nzWidth;
            self.heightRatio = (self.largeHeight/self.currentZoomLevel) / self.nzHeight;
            if(self.options.zoomType == "window") {
                self.zoomWindowStyle = "overflow: hidden;"
                    + "background-position: 0px 0px;text-align:center;"
                    + "background-color: " + String(self.options.zoomWindowBgColour)
                    + ";width: " + String(self.options.zoomWindowWidth) + "px;"
                    + "height: " + String(self.options.zoomWindowHeight)
                    + "px;float: left;"
                    + "background-size: "+ self.largeWidth/self.currentZoomLevel+ "px " +self.largeHeight/self.currentZoomLevel + "px;"
                    + "display: none;z-index:100;"
                    + "border: " + String(self.options.borderSize)
                    + "px solid " + self.options.borderColour
                    + ";background-repeat: no-repeat;"
                    + "position: absolute;";
            }  // if
            if(self.options.zoomType == "inner") {
                var borderWidth = self.$elem.css("border-left-width");
                self.zoomWindowStyle = "overflow: hidden;"
                    + "margin-left: " + String(borderWidth) + ";"
                    + "margin-top: " + String(borderWidth) + ";"
                    + "background-position: 0px 0px;"
                    + "width: " + String(self.nzWidth) + "px;"
                    + "height: " + String(self.nzHeight) + "px;"
                    + "px;float: left;"
                    + "display: none;"
                    + "cursor:"+(self.options.cursor)+";"
                    + "px solid " + self.options.borderColour
                    + ";background-repeat: no-repeat;"
                    + "position: absolute;";
            }  // if
            if(self.options.zoomType == "window") {
                if(self.nzHeight < self.options.zoomWindowWidth/self.widthRatio){
                    lensHeight = self.nzHeight;
                } else {
                    lensHeight = String((self.options.zoomWindowHeight/self.heightRatio))
                } // if-else
                if(self.largeWidth < self.options.zoomWindowWidth){
                    lensWidth = self.nzWidth;
                } else {
                    lensWidth =  (self.options.zoomWindowWidth/self.widthRatio);
                } // if-else
                self.lensStyle = "background-position: 0px 0px;width: " + String((self.options.zoomWindowWidth)/self.widthRatio) + "px;height: " + String((self.options.zoomWindowHeight)/self.heightRatio)
                    + "px;float: right;display: none;"
                    + "overflow: hidden;"
                    + "z-index: 999;"
                    + "-webkit-transform: translateZ(0);"
                    + "opacity:"+(self.options.lensOpacity)+";filter: alpha(opacity = "+(self.options.lensOpacity*100)+"); zoom:1;"
                    + "width:"+lensWidth+"px;"
                    + "height:"+lensHeight+"px;"
                    + "background-color:"+(self.options.lensColour)+";"
                    + "cursor:"+(self.options.cursor)+";"
                    + "border: "+(self.options.lensBorderSize)+"px" +
                    " solid "+(self.options.lensBorderColour)+";background-repeat: no-repeat;position: absolute;";
            }  // if

            //tint style
            self.tintStyle = "display: block;"
                + "position: absolute;"
                + "background-color: "+self.options.tintColour+";"
                + "filter:alpha(opacity=0);"
                + "opacity: 0;"
                + "width: " + self.nzWidth + "px;"
                + "height: " + self.nzHeight + "px;";

            self.lensRound = '';

            if(self.options.zoomType == "lens") {
                self.lensStyle = "background-position: 0px 0px;"
                    + "float: left;display: none;"
                    + "border: " + String(self.options.borderSize) + "px solid " + self.options.borderColour+";"
                    + "width:"+ String(self.options.lensSize) +"px;"
                    + "height:"+ String(self.options.lensSize)+"px;"
                    + "background-repeat: no-repeat;position: absolute;";
            } // if

            if(self.options.lensShape == "round") {
                self.lensRound = "border-top-left-radius: " + String(self.options.lensSize / 2 + self.options.borderSize) + "px;"
                    + "border-top-right-radius: " + String(self.options.lensSize / 2 + self.options.borderSize) + "px;"
                    + "border-bottom-left-radius: " + String(self.options.lensSize / 2 + self.options.borderSize) + "px;"
                    + "border-bottom-right-radius: " + String(self.options.lensSize / 2 + self.options.borderSize) + "px;";
            } // if
            self.zoomContainer = $('<div class="zoomContainer" style="-webkit-transform: translateZ(0);position:absolute;left:'+self.nzOffset.left+'px;top:'+self.nzOffset.top+'px;height:'+self.nzHeight+'px;width:'+self.nzWidth+'px;"></div>');
            $('body').append(self.zoomContainer);
            //this will add overflow hidden and contrain the lens on lens mode
            if(self.options.containLensZoom && self.options.zoomType == "lens"){
                self.zoomContainer.css("overflow", "hidden");
            } // if
            if(self.options.zoomType != "inner") {
                self.zoomLens = $("<div class='zoomLens' style='" + self.lensStyle + self.lensRound +"'>&nbsp;</div>")
                    .appendTo(self.zoomContainer)
                    .click(function () {
                        self.$elem.trigger('click');
                    }); // func
                if(self.options.tint) {
                    self.tintContainer = $('<div/>').addClass('tintContainer');
                    self.zoomTint = $("<div class='zoomTint' style='"+self.tintStyle+"'></div>");
                    self.zoomLens.wrap(self.tintContainer);
                    self.zoomTintcss = self.zoomLens.after(self.zoomTint);
                    self.zoomTintImage = $('<img style="position: absolute; left: 0px; top: 0px; max-width: none; width: '+self.nzWidth+'px; height: '+self.nzHeight+'px;" class="shadow-lg" src="'+self.imageSrc+'">')
                        .appendTo(self.zoomLens)
                        .click(function () {
                            self.$elem.trigger('click');
                        }); // func
                } // if
            } // if
            if(isNaN(self.options.zoomWindowPosition)){
                self.zoomWindow = $("<div style='z-index:999;left:"+(self.windowOffsetLeft)+"px;top:"+(self.windowOffsetTop)+"px;" + self.zoomWindowStyle + "' class='zoomWindow'>&nbsp;</div>")
                    .appendTo('body')
                    .click(function () {
                        self.$elem.trigger('click');
                    });
            }else{
                self.zoomWindow = $("<div style='z-index:999;left:"+(self.windowOffsetLeft)+"px;top:"+(self.windowOffsetTop)+"px;" + self.zoomWindowStyle + "' class='zoomWindow'>&nbsp;</div>")
                    .appendTo(self.zoomContainer)
                    .click(function () {
                        self.$elem.trigger('click');
                    }); // func
            }  // if-else
            self.zoomWindowContainer = $('<div/>').addClass('zoomWindowContainer').css("width",self.options.zoomWindowWidth);
            self.zoomWindow.wrap(self.zoomWindowContainer);
            if(self.options.zoomType == "lens") {
                self.zoomLens.css({ backgroundImage: "url('" + self.imageSrc + "')" });
            } // if
            if(self.options.zoomType == "window") {
                self.zoomWindow.css({ backgroundImage: "url('" + self.imageSrc + "')" });
            } // if
            if(self.options.zoomType == "inner") {
                self.zoomWindow.css({ backgroundImage: "url('" + self.imageSrc + "')" });
            } // if
            self.$elem.bind('touchmove', function(e){
                e.preventDefault();
                var touch = e.originalEvent.touches[0] || e.originalEvent.changedTouches[0];
                self.setPosition(touch);
            });
            self.zoomContainer.bind('touchmove', function(e){
                if(self.options.zoomType == "inner") {
                    self.showHideWindow("show");
                } // if
                e.preventDefault();
                var touch = e.originalEvent.touches[0] || e.originalEvent.changedTouches[0];
                self.setPosition(touch);
            }); // func
            self.zoomContainer.bind('touchend', function(e){
                self.showHideWindow("hide");
                if(self.options.showLens) {self.showHideLens("hide");}
                if(self.options.tint && self.options.zoomType != "inner") {self.showHideTint("hide");}
            }); // func
            self.$elem.bind('touchend', function(e){
                self.showHideWindow("hide");
                if(self.options.showLens) {self.showHideLens("hide");}
                if(self.options.tint && self.options.zoomType != "inner") {self.showHideTint("hide");}
            }); //func
            if(self.options.showLens) {
                self.zoomLens.bind('touchmove', function(e){
                    e.preventDefault();
                    var touch = e.originalEvent.touches[0] || e.originalEvent.changedTouches[0];
                    self.setPosition(touch);
                }); // func
                self.zoomLens.bind('touchend', function(e){
                    self.showHideWindow("hide");
                    if(self.options.showLens) {
                        self.showHideLens("hide");
                    } // if
                    if(self.options.tint && self.options.zoomType != "inner") {
                        self.showHideTint("hide");
                    } // if
                }); // func
            } // if
            self.$elem.bind('mousemove', function(e){
                if(self.overWindow == false){
                    self.setElements("show");
                } // if
                //make sure on orientation change the setposition is not fired
                if(self.lastX !== e.clientX || self.lastY !== e.clientY){
                    self.setPosition(e);
                    self.currentLoc = e;
                }  // if
                self.lastX = e.clientX;
                self.lastY = e.clientY;
            }); // function
            self.zoomContainer.bind('mousemove', function(e){
                if(self.overWindow == false){
                    self.setElements("show");
                } // if
                if(self.lastX !== e.clientX || self.lastY !== e.clientY){
                    self.setPosition(e);
                    self.currentLoc = e;
                } // if
                self.lastX = e.clientX;
                self.lastY = e.clientY;
            }); // func
            if(self.options.zoomType != "inner") {
                self.zoomLens.bind('mousemove', function(e){
                    if(self.lastX !== e.clientX || self.lastY !== e.clientY){
                        self.setPosition(e);
                        self.currentLoc = e;
                    }  // if
                    self.lastX = e.clientX;
                    self.lastY = e.clientY;
                });
            } // if
            if(self.options.tint && self.options.zoomType != "inner") {
                self.zoomTint.bind('mousemove', function(e){
                    if(self.lastX !== e.clientX || self.lastY !== e.clientY){
                        self.setPosition(e);
                        self.currentLoc = e;
                    } // if
                    self.lastX = e.clientX;
                    self.lastY = e.clientY;
                });
            } // if
            if(self.options.zoomType == "inner") {
                self.zoomWindow.bind('mousemove', function(e) {
                    if(self.lastX !== e.clientX || self.lastY !== e.clientY){
                        self.setPosition(e);
                        self.currentLoc = e;
                    } // if
                    self.lastX = e.clientX;
                    self.lastY = e.clientY;
                }); // bind
            } // if
            self.zoomContainer.add(self.$elem).mouseenter(function(){
                if(self.overWindow == false){self.setElements("show");}
            }).mouseleave(function(){
                if(!self.scrollLock){
                    self.setElements("hide");
                    self.options.onDestroy(self.$elem);
                }
            }); // mouseleave
            if(self.options.zoomType != "inner") {
                self.zoomWindow.mouseenter(function(){
                    self.overWindow = true;
                    self.setElements("hide");
                }).mouseleave(function(){
                    self.overWindow = false;
                }); // mouseleave
            } // if

            if (self.options.zoomLevel != 1){
                //  self.changeZoomLevel(self.currentZoomLevel);
            } // if
            if(self.options.minZoomLevel){
                self.minZoomLevel = self.options.minZoomLevel;
            } else{
                self.minZoomLevel = self.options.scrollZoomIncrement * 2;
            } // if-else
            if(self.options.scrollZoom){
                self.zoomContainer.add(self.$elem).bind('mousewheel DOMMouseScroll MozMousePixelScroll', function(e){
                    self.scrollLock = true;
                    clearTimeout($.data(this, 'timer'));
                    $.data(this, 'timer', setTimeout(function() {
                        self.scrollLock = false;
                    }, 250));
                    var theEvent = e.originalEvent.wheelDelta || e.originalEvent.detail*-1; // decent tv show
                    e.stopImmediatePropagation();
                    e.stopPropagation();
                    e.preventDefault();
                    if(theEvent /120 > 0) {
                        if(self.currentZoomLevel >= self.minZoomLevel){
                            self.changeZoomLevel(self.currentZoomLevel-self.options.scrollZoomIncrement);
                        } // if
                    } else {
                        if(self.options.maxZoomLevel){
                            if(self.currentZoomLevel <= self.options.maxZoomLevel){
                                self.changeZoomLevel(parseFloat(self.currentZoomLevel)+self.options.scrollZoomIncrement);
                            } // if
                        } else{
                            self.changeZoomLevel(parseFloat(self.currentZoomLevel)+self.options.scrollZoomIncrement);
                        } // if-else
                    } // if-else
                    return false;
                }); //func
            } // if
        }, // func
        setElements: function(type) {
            var self = this;
            if(!self.options.zoomEnabled){return false;}
            if(type=="show"){
                if(self.isWindowSet){
                    if(self.options.zoomType == "inner") {
                        self.showHideWindow("show");
                    } // if
                    if(self.options.zoomType == "window") {
                        self.showHideWindow("show");
                    } // if
                    if(self.options.showLens) {
                        self.showHideLens("show");
                    } // if
                    if(self.options.tint && self.options.zoomType != "inner") {
                        self.showHideTint("show");
                    } // if
                } // if
            } // if
            if(type=="hide"){
                if(self.options.zoomType == "window") {self.showHideWindow("hide");}
                if(!self.options.tint) {self.showHideWindow("hide");}
                if(self.options.showLens) {self.showHideLens("hide");}
                if(self.options.tint) { self.showHideTint("hide");}
            } // if
        }, // func
        setPosition: function(e) {
            var self = this;
            if(!self.options.zoomEnabled){
                return false;
            } // if
            self.nzHeight = self.$elem.height();
            self.nzWidth = self.$elem.width();
            self.nzOffset = self.$elem.offset();
            self.zoomLens.addClass("shadow-lg");
            if(self.options.tint && self.options.zoomType != "inner") {
                self.zoomTint.css({ top: 0});
                self.zoomTint.css({ left: 0});
            } // if
            if(self.options.responsive && !self.options.scrollZoom){
                if(self.options.showLens){
                    if(self.nzHeight < self.options.zoomWindowWidth/self.widthRatio){
                        lensHeight = self.nzHeight;
                    } else {
                        lensHeight = String((self.options.zoomWindowHeight/self.heightRatio))
                    } // if-else
                    if(self.largeWidth < self.options.zoomWindowWidth){
                        lensWidth = self.nzWidth;
                    } else {
                        lensWidth =  (self.options.zoomWindowWidth/self.widthRatio);
                    } // if-else
                    self.widthRatio = self.largeWidth / self.nzWidth;
                    self.heightRatio = self.largeHeight / self.nzHeight;
                    if(self.options.zoomType != "lens") {
                        if(self.nzHeight < self.options.zoomWindowWidth/self.widthRatio){
                            lensHeight = self.nzHeight;
                        } else{
                            lensHeight = String((self.options.zoomWindowHeight/self.heightRatio))
                        } // if-else
                        if(self.nzWidth < self.options.zoomWindowHeight/self.heightRatio){
                            lensWidth = self.nzWidth;
                        } else{
                            lensWidth =  String((self.options.zoomWindowWidth/self.widthRatio));
                        } // if-else
                        self.zoomLens.css('width', lensWidth);
                        self.zoomLens.css('height', lensHeight);
                        if(self.options.tint){
                            self.zoomTintImage.css('width', self.nzWidth);
                            self.zoomTintImage.css('height', self.nzHeight);
                        } // if
                    } // if
                    if(self.options.zoomType == "lens") {
                        self.zoomLens.css({ width: String(self.options.lensSize) + 'px', height: String(self.options.lensSize) + 'px' })
                    } // if
                } // if
            } // if
            self.zoomContainer.css({ top: self.nzOffset.top});
            self.zoomContainer.css({ left: self.nzOffset.left});
            self.mouseLeft = parseInt(e.pageX - self.nzOffset.left);
            self.mouseTop = parseInt(e.pageY - self.nzOffset.top);
            if(self.options.zoomType == "window") {
                self.Etoppos = (self.mouseTop < (self.zoomLens.height()/2));
                self.Eboppos = (self.mouseTop > self.nzHeight - (self.zoomLens.height()/2)-(self.options.lensBorderSize*2));
                self.Eloppos = (self.mouseLeft < 0+((self.zoomLens.width()/2)));
                self.Eroppos = (self.mouseLeft > (self.nzWidth - (self.zoomLens.width()/2)-(self.options.lensBorderSize*2)));
            } // if
            if(self.options.zoomType == "inner"){
                self.Etoppos = (self.mouseTop < ((self.nzHeight/2)/self.heightRatio) );
                self.Eboppos = (self.mouseTop > (self.nzHeight - ((self.nzHeight/2)/self.heightRatio)));
                self.Eloppos = (self.mouseLeft < 0+(((self.nzWidth/2)/self.widthRatio)));
                self.Eroppos = (self.mouseLeft > (self.nzWidth - (self.nzWidth/2)/self.widthRatio-(self.options.lensBorderSize*2)));
            } // if

            // if the mouse position of the slider is one of the outerbounds, then hide  window and lens
            if (self.mouseLeft < 0 || self.mouseTop < 0 || self.mouseLeft > self.nzWidth || self.mouseTop > self.nzHeight ) {
                self.setElements("hide");
                return;
            } else {
                //lens options
                if(self.options.showLens) {
                    self.lensLeftPos = String(Math.floor(self.mouseLeft - self.zoomLens.width() / 2));
                    self.lensTopPos = String(Math.floor(self.mouseTop - self.zoomLens.height() / 2));
                } // if

                //Top region
                if(self.Etoppos){
                    self.lensTopPos = 0;
                } // if
                //Left Region
                if(self.Eloppos){
                    self.windowLeftPos = 0;
                    self.lensLeftPos = 0;
                    self.tintpos=0;
                } // if
                if(self.options.zoomType == "window") {
                    if(self.Eboppos){
                        self.lensTopPos = Math.max( (self.nzHeight)-self.zoomLens.height()-(self.options.lensBorderSize*2), 0 );
                    } // if
                    if(self.Eroppos){
                        self.lensLeftPos = (self.nzWidth-(self.zoomLens.width())-(self.options.lensBorderSize*2));
                    }  // if
                } // if
                if(self.options.zoomType == "inner") {
                    if(self.Eboppos){
                        self.lensTopPos = Math.max( ((self.nzHeight)-(self.options.lensBorderSize*2)), 0 );
                    } // if
                    if(self.Eroppos){
                        self.lensLeftPos = (self.nzWidth-(self.nzWidth)-(self.options.lensBorderSize*2));
                    }  // if
                } // if
                if(self.options.zoomType == "lens") {
                    self.windowLeftPos = String(((e.pageX - self.nzOffset.left) * self.widthRatio - self.zoomLens.width() / 2) * (-1));
                    self.windowTopPos = String(((e.pageY - self.nzOffset.top) * self.heightRatio - self.zoomLens.height() / 2) * (-1));
                    self.zoomLens.css({ backgroundPosition: self.windowLeftPos + 'px ' + self.windowTopPos + 'px' });
                    if(self.changeBgSize){
                        if(self.nzHeight>self.nzWidth){
                            if(self.options.zoomType == "lens"){
                                self.zoomLens.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                            } // if
                            self.zoomWindow.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                        } else {
                            if(self.options.zoomType == "lens"){
                                self.zoomLens.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                            } // if
                            self.zoomWindow.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                        } // if-else
                        self.changeBgSize = false;
                    } // if
                    self.setWindowPostition(e);
                } // if

                if(self.options.tint && self.options.zoomType != "inner") {
                    self.setTintPosition(e);
                } // if
                if(self.options.zoomType == "window") {
                    self.setWindowPostition(e);
                } // if
                if(self.options.zoomType == "inner") {
                    self.setWindowPostition(e);
                } // if
                if(self.options.showLens) {
                    if(self.fullwidth && self.options.zoomType != "lens"){
                        self.lensLeftPos = 0;
                    } // if
                    self.zoomLens.css({ left: self.lensLeftPos + 'px', top: self.lensTopPos + 'px' })
                } // if
            } // if-else
        }, // func
        showHideWindow: function(change) {
            var self = this;
            if(change == "show"){
                if(!self.isWindowActive){
                    if(self.options.zoomWindowFadeIn){
                        self.zoomWindow.stop(true, true, false).fadeIn(self.options.zoomWindowFadeIn);
                    } else {
                        self.zoomWindow.show();
                    } // if-else
                    self.isWindowActive = true;
                } // if
            } // if
            if(change == "hide"){
                if(self.isWindowActive){
                    if(self.options.zoomWindowFadeOut){
                        self.zoomWindow.stop(true, true).fadeOut(self.options.zoomWindowFadeOut, function () {
                            if (self.loop) {
                                clearInterval(self.loop);
                                self.loop = false;
                            } // if
                        }); // func
                    } else {
                        self.zoomWindow.hide();
                    } // if-else
                    self.isWindowActive = false;
                }  // if
            } // if
        }, // func
        showHideLens: function(change) {
            var self = this;
            if(change == "show"){
                if(!self.isLensActive){
                    if(self.options.lensFadeIn){
                        self.zoomLens.stop(true, true, false).fadeIn(self.options.lensFadeIn);
                    } else{
                        self.zoomLens.show();
                    } // if-else
                    self.isLensActive = true;
                } // if
            } // if
            if(change == "hide"){
                if(self.isLensActive){
                    if(self.options.lensFadeOut){
                        self.zoomLens.stop(true, true).fadeOut(self.options.lensFadeOut);
                    } else{
                        self.zoomLens.hide();
                    } // if-else
                    self.isLensActive = false;
                }  // if
            } // if
        }, // func
        showHideTint: function(change) {
            var self = this;
            if(change == "show"){
                if(!self.isTintActive){
                    if(self.options.zoomTintFadeIn){
                        self.zoomTint.css({opacity:self.options.tintOpacity}).animate().stop(true, true).fadeIn("slow");
                    } else{
                        self.zoomTint.css({opacity:self.options.tintOpacity}).animate();
                        self.zoomTint.show();
                    } // if -else
                    self.isTintActive = true;
                } // if
            } // if
            if(change == "hide"){
                if(self.isTintActive){

                    if(self.options.zoomTintFadeOut){
                        self.zoomTint.stop(true, true).fadeOut(self.options.zoomTintFadeOut);
                    } else {
                        self.zoomTint.hide();
                    } // if-else
                    self.isTintActive = false;
                } // if
            } // if
        }, // func
        setLensPostition: function( e ) {
        }, // func
        setWindowPostition: function( e ) {
            var self = this;
            if(!isNaN(self.options.zoomWindowPosition)){
                switch (self.options.zoomWindowPosition) {
                    case 1: //done
                        self.windowOffsetTop = (self.options.zoomWindowOffety);//DONE - 1
                        self.windowOffsetLeft =(+self.nzWidth); //DONE 1, 2, 3, 4, 16
                        break;
                    case 2:
                        if(self.options.zoomWindowHeight > self.nzHeight){ //positive margin
                            self.windowOffsetTop = ((self.options.zoomWindowHeight/2)-(self.nzHeight/2))*(-1);
                            self.windowOffsetLeft =(self.nzWidth); //DONE 1, 2, 3, 4, 16
                        } // if
                        break;
                    case 3: //done
                        self.windowOffsetTop = (self.nzHeight - self.zoomWindow.height() - (self.options.borderSize*2)); //DONE 3,9
                        self.windowOffsetLeft =(self.nzWidth); //DONE 1, 2, 3, 4, 16
                        break;
                    case 4: //done
                        self.windowOffsetTop = (self.nzHeight); //DONE - 4,5,6,7,8
                        self.windowOffsetLeft =(self.nzWidth); //DONE 1, 2, 3, 4, 16
                        break;
                    case 5: //done
                        self.windowOffsetTop = (self.nzHeight); //DONE - 4,5,6,7,8
                        self.windowOffsetLeft =(self.nzWidth-self.zoomWindow.width()-(self.options.borderSize*2)); //DONE - 5,15
                        break;
                    case 6:
                        if(self.options.zoomWindowHeight > self.nzHeight){ //positive margin
                            self.windowOffsetTop = (self.nzHeight);  //DONE - 4,5,6,7,8
                            self.windowOffsetLeft =((self.options.zoomWindowWidth/2)-(self.nzWidth/2)+(self.options.borderSize*2))*(-1);
                        } // if
                        break;
                    case 7: //done
                        self.windowOffsetTop = (self.nzHeight);  //DONE - 4,5,6,7,8
                        self.windowOffsetLeft = 0; //DONE 7, 13
                        break;
                    case 8: //done
                        self.windowOffsetTop = (self.nzHeight); //DONE - 4,5,6,7,8
                        self.windowOffsetLeft =(self.zoomWindow.width()+(self.options.borderSize*2) )* (-1);  //DONE 8,9,10,11,12
                        break;
                    case 9:  //done
                        self.windowOffsetTop = (self.nzHeight - self.zoomWindow.height() - (self.options.borderSize*2)); //DONE 3,9
                        self.windowOffsetLeft =(self.zoomWindow.width()+(self.options.borderSize*2) )* (-1);  //DONE 8,9,10,11,12
                        break;
                    case 10:
                        if(self.options.zoomWindowHeight > self.nzHeight){ //positive margin
                            self.windowOffsetTop = ((self.options.zoomWindowHeight/2)-(self.nzHeight/2))*(-1);
                            self.windowOffsetLeft =(self.zoomWindow.width()+(self.options.borderSize*2) )* (-1);  //DONE 8,9,10,11,12
                        } // if
                        break;
                    case 11:
                        self.windowOffsetTop = (self.options.zoomWindowOffety);
                        self.windowOffsetLeft =(self.zoomWindow.width()+(self.options.borderSize*2) )* (-1);  //DONE 8,9,10,11,12
                        break;
                    case 12: //done
                        self.windowOffsetTop = (self.zoomWindow.height()+(self.options.borderSize*2))*(-1); //DONE 12,13,14,15,16
                        self.windowOffsetLeft =(self.zoomWindow.width()+(self.options.borderSize*2) )* (-1);  //DONE 8,9,10,11,12
                        break;
                    case 13: //done
                        self.windowOffsetTop = (self.zoomWindow.height()+(self.options.borderSize*2))*(-1); //DONE 12,13,14,15,16
                        self.windowOffsetLeft =(0); //DONE 7, 13
                        break;
                    case 14:
                        if(self.options.zoomWindowHeight > self.nzHeight){ //positive margin
                            self.windowOffsetTop = (self.zoomWindow.height()+(self.options.borderSize*2))*(-1); //DONE 12,13,14,15,16
                            self.windowOffsetLeft =((self.options.zoomWindowWidth/2)-(self.nzWidth/2)+(self.options.borderSize*2))*(-1);
                        } // if
                        break;
                    case 15://done
                        self.windowOffsetTop = (self.zoomWindow.height()+(self.options.borderSize*2))*(-1); //DONE 12,13,14,15,16
                        self.windowOffsetLeft =(self.nzWidth-self.zoomWindow.width()-(self.options.borderSize*2)); //DONE - 5,15
                        break;
                    case 16:  //done
                        self.windowOffsetTop = (self.zoomWindow.height()+(self.options.borderSize*2))*(-1); //DONE 12,13,14,15,16
                        self.windowOffsetLeft =(self.nzWidth); //DONE 1, 2, 3, 4, 16
                        break;
                    default: //done
                        self.windowOffsetTop = (self.options.zoomWindowOffety);//DONE - 1
                        self.windowOffsetLeft =(self.nzWidth); //DONE 1, 2, 3, 4, 16
                } // case
            } else {
                //WE CAN POSITION IN A CLASS - ASSUME THAT ANY STRING PASSED IS
                self.externalContainer = $('#'+self.options.zoomWindowPosition);
                self.externalContainerWidth = self.externalContainer.width();
                self.externalContainerHeight = self.externalContainer.height();
                self.externalContainerOffset = self.externalContainer.offset();
                self.windowOffsetTop = self.externalContainerOffset.top;//DONE - 1
                self.windowOffsetLeft =self.externalContainerOffset.left; //DONE 1, 2, 3, 4, 16
            } // if-else
            self.isWindowSet = true;
            self.windowOffsetTop = self.windowOffsetTop + self.options.zoomWindowOffety;
            self.windowOffsetLeft = self.windowOffsetLeft + self.options.zoomWindowOffetx;
            self.zoomWindow.css({ top: self.windowOffsetTop});
            self.zoomWindow.css({ left: self.windowOffsetLeft});
            if(self.options.zoomType == "inner") {
                self.zoomWindow.css({ top: 0});
                self.zoomWindow.css({ left: 0});
            } // if
            self.windowLeftPos = String(((e.pageX - self.nzOffset.left) * self.widthRatio - self.zoomWindow.width() / 2) * (-1));
            self.windowTopPos = String(((e.pageY - self.nzOffset.top) * self.heightRatio - self.zoomWindow.height() / 2) * (-1));
            if(self.Etoppos){self.windowTopPos = 0;}
            if(self.Eloppos){self.windowLeftPos = 0;}
            if(self.Eboppos){self.windowTopPos = (self.largeHeight/self.currentZoomLevel-self.zoomWindow.height())*(-1);  }
            if(self.Eroppos){self.windowLeftPos = ((self.largeWidth/self.currentZoomLevel-self.zoomWindow.width())*(-1));}
            if(self.fullheight){
                self.windowTopPos = 0;
            } // if
            if(self.fullwidth){
                self.windowLeftPos = 0;
            } // if
            if(self.options.zoomType == "window" || self.options.zoomType == "inner") {
                if(self.zoomLock == 1){
                    //overrides for images not zoomable
                    if(self.widthRatio <= 1){
                        self.windowLeftPos = 0;
                    } // if
                    if(self.heightRatio <= 1){
                        self.windowTopPos = 0;
                    } // if
                } // if
                if (self.options.zoomType == "window") {
                    if (self.largeHeight < self.options.zoomWindowHeight) {
                        self.windowTopPos = 0;
                    } // if
                    if (self.largeWidth < self.options.zoomWindowWidth) {
                        self.windowLeftPos = 0;
                    } // if
                } // if
                if (self.options.easing){
                    if(!self.xp){self.xp = 0;}
                    if(!self.yp){self.yp = 0;}
                    if (!self.loop){
                        self.loop = setInterval(function(){
                            self.xp += (self.windowLeftPos  - self.xp) / self.options.easingAmount;
                            self.yp += (self.windowTopPos  - self.yp) / self.options.easingAmount;
                            if(self.scrollingLock){
                                clearInterval(self.loop);
                                self.xp = self.windowLeftPos;
                                self.yp = self.windowTopPos;
                                self.xp = ((e.pageX - self.nzOffset.left) * self.widthRatio - self.zoomWindow.width() / 2) * (-1);
                                self.yp = (((e.pageY - self.nzOffset.top) * self.heightRatio - self.zoomWindow.height() / 2) * (-1));
                                if(self.changeBgSize){
                                    if(self.nzHeight>self.nzWidth){
                                        if(self.options.zoomType == "lens"){
                                            self.zoomLens.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                                        } // if
                                        self.zoomWindow.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                                    } else {
                                        if(self.options.zoomType != "lens"){
                                            self.zoomLens.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                                        } // if
                                        self.zoomWindow.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                                    } // if-else
                                    self.changeBgSize = false;
                                } // if
                                self.zoomWindow.css({ backgroundPosition: self.windowLeftPos + 'px ' + self.windowTopPos + 'px' });
                                self.scrollingLock = false;
                                self.loop = false;
                            } else if (Math.round(Math.abs(self.xp - self.windowLeftPos) + Math.abs(self.yp - self.windowTopPos)) < 1) {
                                clearInterval(self.loop);
                                self.zoomWindow.css({ backgroundPosition: self.windowLeftPos + 'px ' + self.windowTopPos + 'px' });
                                self.loop = false;
                            } else {
                                if(self.changeBgSize){
                                    if(self.nzHeight>self.nzWidth){
                                        if(self.options.zoomType == "lens"){
                                            self.zoomLens.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                                        } // if
                                        self.zoomWindow.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                                    } else {
                                        if(self.options.zoomType != "lens"){
                                            self.zoomLens.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                                        } // if
                                        self.zoomWindow.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                                    } // if-else
                                    self.changeBgSize = false;
                                } // if
                                self.zoomWindow.css({ backgroundPosition: self.xp + 'px ' + self.yp + 'px' });
                            } // if-else
                        }, 16);
                    } // if
                } else {
                    if(self.changeBgSize){
                        if(self.nzHeight>self.nzWidth){
                            if(self.options.zoomType == "lens"){
                                self.zoomLens.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                            } // if
                            self.zoomWindow.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                        } else {
                            if(self.options.zoomType == "lens"){
                                self.zoomLens.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                            }
                            if((self.largeHeight/self.newvaluewidth) < self.options.zoomWindowHeight){

                                self.zoomWindow.css({ "background-size": self.largeWidth/self.newvaluewidth + 'px ' + self.largeHeight/self.newvaluewidth + 'px' });
                            } else {
                                self.zoomWindow.css({ "background-size": self.largeWidth/self.newvalueheight + 'px ' + self.largeHeight/self.newvalueheight + 'px' });
                            } // if-else
                        } // if-else
                        self.changeBgSize = false;
                    } // if-else
                    self.zoomWindow.css({ backgroundPosition: self.windowLeftPos + 'px ' + self.windowTopPos + 'px' });
                } // if-elseif-else
            }
        }, // func
        setTintPosition: function(e){
            var self = this;
            self.nzOffset = self.$elem.offset();
            self.tintpos = String(((e.pageX - self.nzOffset.left)-(self.zoomLens.width() / 2)) * (-1));
            self.tintposy = String(((e.pageY - self.nzOffset.top) - self.zoomLens.height() / 2) * (-1));
            if(self.Etoppos){
                self.tintposy = 0;
            } // if
            if(self.Eloppos){
                self.tintpos=0;
            } // if
            if(self.Eboppos){
                self.tintposy = (self.nzHeight-self.zoomLens.height()-(self.options.lensBorderSize*2))*(-1);
            } // if
            if(self.Eroppos){
                self.tintpos = ((self.nzWidth-self.zoomLens.width()-(self.options.lensBorderSize*2))*(-1));
            } // if
            if(self.options.tint) {
                if(self.fullheight){
                    self.tintposy = 0;
                } // if
                if(self.fullwidth){
                    self.tintpos = 0;
                } // if
                self.zoomTintImage.css({'left': self.tintpos+'px'});
                self.zoomTintImage.css({'top': self.tintposy+'px'});
            } // if
        }, // func
        swaptheimage: function(smallimage, largeimage){
            var self = this;
            var newImg = new Image();
            if(self.options.loadingIcon){
                self.spinner = $('<div style="background: url(\''+self.options.loadingIcon+'\') no-repeat center;height:'+self.nzHeight+'px;width:'+self.nzWidth+'px;z-index: 2000;position: absolute; background-position: center center;"></div>');
                self.$elem.after(self.spinner);
            } // if
            self.options.onImageSwap(self.$elem);
            newImg.onload = function() {
                self.largeWidth = newImg.width;
                self.largeHeight = newImg.height;
                self.zoomImage = largeimage;
                self.zoomWindow.css({ "background-size": self.largeWidth + 'px ' + self.largeHeight + 'px' });
                self.swapAction(smallimage, largeimage);
                return;
            } // func
            newImg.src = largeimage; // this must be done AFTER setting onload
        }, // func
        swapAction: function(smallimage, largeimage){
            var self = this;
            var newImg2 = new Image();
            newImg2.onload = function() {
                self.nzHeight = newImg2.height;
                self.nzWidth = newImg2.width;
                self.options.onImageSwapComplete(self.$elem);
                self.doneCallback();
                return;
            } // func
            newImg2.src = smallimage;
            self.currentZoomLevel = self.options.zoomLevel;
            self.options.maxZoomLevel = false;
            if(self.options.zoomType == "lens") {
                self.zoomLens.css({ backgroundImage: "url('" + largeimage + "')" });
            } // if
            if(self.options.zoomType == "window") {
                self.zoomWindow.css({ backgroundImage: "url('" + largeimage + "')" });
            } // if
            if(self.options.zoomType == "inner") {
                self.zoomWindow.css({ backgroundImage: "url('" + largeimage + "')" });
            } // if

            self.currentImage = largeimage;
            if(self.options.imageCrossfade){
                var oldImg = self.$elem;
                var newImg = oldImg.clone();
                self.$elem.attr("src",smallimage)
                self.$elem.after(newImg);
                newImg.stop(true).fadeOut(self.options.imageCrossfade, function() {
                    $(this).remove();
                }); // stop
                self.$elem.width("auto").removeAttr("width");
                self.$elem.height("auto").removeAttr("height");
                oldImg.fadeIn(self.options.imageCrossfade);
                if(self.options.tint && self.options.zoomType != "inner") {
                    var oldImgTint = self.zoomTintImage;
                    var newImgTint = oldImgTint.clone();
                    self.zoomTintImage.attr("src",largeimage)
                    self.zoomTintImage.after(newImgTint);
                    newImgTint.stop(true).fadeOut(self.options.imageCrossfade, function() {
                        $(this).remove();
                    }); // stop
                    oldImgTint.fadeIn(self.options.imageCrossfade);
                    self.zoomTint.css({ height: self.$elem.height()});
                    self.zoomTint.css({ width: self.$elem.width()});
                } // if
                self.zoomContainer.css("height", self.$elem.height());
                self.zoomContainer.css("width", self.$elem.width());
                if(self.options.zoomType == "inner"){
                    if(!self.options.constrainType){
                        self.zoomWrap.parent().css("height", self.$elem.height());
                        self.zoomWrap.parent().css("width", self.$elem.width());
                        self.zoomWindow.css("height", self.$elem.height());
                        self.zoomWindow.css("width", self.$elem.width());
                    } // if
                } // if
                if(self.options.imageCrossfade){
                    self.zoomWrap.css("height", self.$elem.height());
                    self.zoomWrap.css("width", self.$elem.width());
                } // if
            } else {
                self.$elem.attr("src",smallimage);
                if(self.options.tint) {
                    self.zoomTintImage.attr("src",largeimage);
                    self.zoomTintImage.attr("height",self.$elem.height());
                    self.zoomTintImage.css({ height: self.$elem.height()});
                    self.zoomTint.css({ height: self.$elem.height()});
                } // if
                self.zoomContainer.css("height", self.$elem.height());
                self.zoomContainer.css("width", self.$elem.width());
                if(self.options.imageCrossfade){
                    self.zoomWrap.css("height", self.$elem.height());
                    self.zoomWrap.css("width", self.$elem.width());
                }  // if
            } // if-else
            if(self.options.constrainType){
                if(self.options.constrainType == "height"){
                    self.zoomContainer.css("height", self.options.constrainSize);
                    self.zoomContainer.css("width", "auto");
                    if(self.options.imageCrossfade){
                        self.zoomWrap.css("height", self.options.constrainSize);
                        self.zoomWrap.css("width", "auto");
                        self.constwidth = self.zoomWrap.width();
                    } else {
                        self.$elem.css("height", self.options.constrainSize);
                        self.$elem.css("width", "auto");
                        self.constwidth = self.$elem.width();
                    } // if-else
                    if(self.options.zoomType == "inner"){
                        self.zoomWrap.parent().css("height", self.options.constrainSize);
                        self.zoomWrap.parent().css("width", self.constwidth);
                        self.zoomWindow.css("height", self.options.constrainSize);
                        self.zoomWindow.css("width", self.constwidth);
                    } // if
                    if(self.options.tint){
                        self.tintContainer.css("height", self.options.constrainSize);
                        self.tintContainer.css("width", self.constwidth);
                        self.zoomTint.css("height", self.options.constrainSize);
                        self.zoomTint.css("width", self.constwidth);
                        self.zoomTintImage.css("height", self.options.constrainSize);
                        self.zoomTintImage.css("width", self.constwidth);
                    } // if
                }// if
                if(self.options.constrainType == "width"){
                    self.zoomContainer.css("height", "auto");
                    self.zoomContainer.css("width", self.options.constrainSize);
                    if(self.options.imageCrossfade){
                        self.zoomWrap.css("height", "auto");
                        self.zoomWrap.css("width", self.options.constrainSize);
                        self.constheight = self.zoomWrap.height();
                    } else {
                        self.$elem.css("height", "auto");
                        self.$elem.css("width", self.options.constrainSize);
                        self.constheight = self.$elem.height();
                    } // if-else
                    if(self.options.zoomType == "inner"){
                        self.zoomWrap.parent().css("height", self.constheight);
                        self.zoomWrap.parent().css("width", self.options.constrainSize);
                        self.zoomWindow.css("height", self.constheight);
                        self.zoomWindow.css("width", self.options.constrainSize);
                    } // if
                    if(self.options.tint){
                        self.tintContainer.css("height", self.constheight);
                        self.tintContainer.css("width", self.options.constrainSize);
                        self.zoomTint.css("height", self.constheight);
                        self.zoomTint.css("width", self.options.constrainSize);
                        self.zoomTintImage.css("height", self.constheight);
                        self.zoomTintImage.css("width", self.options.constrainSize);
                    } // if
                } // if
            } // if
        }, // func
        doneCallback: function(){
            var self = this;
            if(self.options.loadingIcon){
                self.spinner.hide();
            } // if
            self.nzOffset = self.$elem.offset();
            self.nzWidth = self.$elem.width();
            self.nzHeight = self.$elem.height();
            self.currentZoomLevel = self.options.zoomLevel;
            self.widthRatio = self.largeWidth / self.nzWidth;
            self.heightRatio = self.largeHeight / self.nzHeight;

            if(self.options.zoomType == "window") {
                if(self.nzHeight < self.options.zoomWindowWidth/self.widthRatio){
                    lensHeight = self.nzHeight;
                }else{
                    lensHeight = String((self.options.zoomWindowHeight/self.heightRatio))
                } // if-else
                if(self.options.zoomWindowWidth < self.options.zoomWindowWidth){
                    lensWidth = self.nzWidth;
                }else{
                    lensWidth =  (self.options.zoomWindowWidth/self.widthRatio);
                } // if-else
                if(self.zoomLens){
                    self.zoomLens.css('width', lensWidth);
                    self.zoomLens.css('height', lensHeight);
                } // if
            } // if
        }, // func
        getCurrentImage: function(){
            var self = this;
            return self.zoomImage;
        }, // func
        getGalleryList: function(){
            var self = this;
            self.gallerylist = [];
            if (self.options.gallery){
                $('#'+self.options.gallery + ' a').each(function() {
                    var img_src = '';
                    if($(this).data("zoom-image")){
                        img_src = $(this).data("zoom-image");
                    } else if($(this).data("image")){
                        img_src = $(this).data("image");
                    } // if-else if
                    if(img_src == self.zoomImage){
                        self.gallerylist.unshift({
                            href: ''+img_src+'',
                            title: $(this).find('img').attr("title")
                        });
                    } else {
                        self.gallerylist.push({
                            href: ''+img_src+'',
                            title: $(this).find('img').attr("title")
                        }); // push
                    } // if-else
                }); // func
            } else {
                self.gallerylist.push({
                    href: ''+self.zoomImage+'',
                    title: $(this).find('img').attr("title")
                });  // push
            } // if-else
            return self.gallerylist;
        }, // func
        changeZoomLevel: function(value){
            var self = this;
            self.scrollingLock = true;
            self.newvalue = parseFloat(value).toFixed(2);
            newvalue = parseFloat(value).toFixed(2);
            maxheightnewvalue = self.largeHeight/((self.options.zoomWindowHeight / self.nzHeight) * self.nzHeight);
            maxwidthtnewvalue = self.largeWidth/((self.options.zoomWindowWidth / self.nzWidth) * self.nzWidth);

            if(self.options.zoomType != "inner"){
                if(maxheightnewvalue <= newvalue){
                    self.heightRatio = (self.largeHeight/maxheightnewvalue) / self.nzHeight;
                    self.newvalueheight = maxheightnewvalue;
                    self.fullheight = true;
                }else{
                    self.heightRatio = (self.largeHeight/newvalue) / self.nzHeight;
                    self.newvalueheight = newvalue;
                    self.fullheight = false;
                } // if-else
                if(maxwidthtnewvalue <= newvalue){
                    self.widthRatio = (self.largeWidth/maxwidthtnewvalue) / self.nzWidth;
                    self.newvaluewidth = maxwidthtnewvalue;
                    self.fullwidth = true;

                }else{
                    self.widthRatio = (self.largeWidth/newvalue) / self.nzWidth;
                    self.newvaluewidth = newvalue;
                    self.fullwidth = false;
                } // if-else
                if(self.options.zoomType == "lens"){
                    if(maxheightnewvalue <= newvalue){
                        self.fullwidth = true;
                        self.newvaluewidth = maxheightnewvalue;
                    } else{
                        self.widthRatio = (self.largeWidth/newvalue) / self.nzWidth;
                        self.newvaluewidth = newvalue;
                        self.fullwidth = false;
                    } // if-else
                } // if
            } // if



            if(self.options.zoomType == "inner") {
                maxheightnewvalue = parseFloat(self.largeHeight/self.nzHeight).toFixed(2);
                maxwidthtnewvalue = parseFloat(self.largeWidth/self.nzWidth).toFixed(2);
                if(newvalue > maxheightnewvalue){
                    newvalue = maxheightnewvalue;
                } // if
                if(newvalue > maxwidthtnewvalue){
                    newvalue = maxwidthtnewvalue;
                } // if

                if(maxheightnewvalue <= newvalue){
                    self.heightRatio = (self.largeHeight/newvalue) / self.nzHeight;
                    if(newvalue > maxheightnewvalue){
                        self.newvalueheight = maxheightnewvalue;
                    }else{
                        self.newvalueheight = newvalue;
                    } //if-else
                    self.fullheight = true;
                }else{
                    self.heightRatio = (self.largeHeight/newvalue) / self.nzHeight;
                    if(newvalue > maxheightnewvalue){
                        self.newvalueheight = maxheightnewvalue;
                    }else{
                        self.newvalueheight = newvalue;
                    } // if-else
                    self.fullheight = false;
                } // if-else

                if(maxwidthtnewvalue <= newvalue){
                    self.widthRatio = (self.largeWidth/newvalue) / self.nzWidth;
                    if(newvalue > maxwidthtnewvalue){
                        self.newvaluewidth = maxwidthtnewvalue;
                    }else{
                        self.newvaluewidth = newvalue;
                    } // if-else
                    self.fullwidth = true;
                }else{
                    self.widthRatio = (self.largeWidth/newvalue) / self.nzWidth;
                    self.newvaluewidth = newvalue;
                    self.fullwidth = false;
                } // if-else
            } // if
            scrcontinue = false;

            if(self.options.zoomType == "inner"){
                if(self.nzWidth >= self.nzHeight){
                    if( self.newvaluewidth <= maxwidthtnewvalue){
                        scrcontinue = true;
                    } else {
                        scrcontinue = false;
                        self.fullheight = true;
                        self.fullwidth = true;
                    } // if-else
                } // if
                if(self.nzHeight > self.nzWidth){
                    if( self.newvaluewidth <= maxwidthtnewvalue){
                        scrcontinue = true;
                    } else {
                        scrcontinue = false;

                        self.fullheight = true;
                        self.fullwidth = true;
                    } // if-else
                } // if
            } // if

            if(self.options.zoomType != "inner"){
                scrcontinue = true;
            } // if

            if(scrcontinue){
                self.zoomLock = 0;
                self.changeZoom = true;
                if(((self.options.zoomWindowHeight)/self.heightRatio) <= self.nzHeight){
                    self.currentZoomLevel = self.newvalueheight;
                    if(self.options.zoomType != "lens" && self.options.zoomType != "inner") {
                        self.changeBgSize = true;
                        self.zoomLens.css({height: String((self.options.zoomWindowHeight)/self.heightRatio) + 'px' })
                    } // if
                    if(self.options.zoomType == "lens" || self.options.zoomType == "inner") {
                        self.changeBgSize = true;
                    } //if
                } // if

                if((self.options.zoomWindowWidth/self.widthRatio) <= self.nzWidth){
                    if(self.options.zoomType != "inner"){
                        if(self.newvaluewidth > self.newvalueheight) {
                            self.currentZoomLevel = self.newvaluewidth;
                        } // if
                    } // if

                    if(self.options.zoomType != "lens" && self.options.zoomType != "inner") {
                        self.changeBgSize = true;
                        self.zoomLens.css({width: String((self.options.zoomWindowWidth)/self.widthRatio) + 'px' })
                    } // if
                    if(self.options.zoomType == "lens" || self.options.zoomType == "inner") {
                        self.changeBgSize = true;
                    } // if
                } // if
                if(self.options.zoomType == "inner"){
                    self.changeBgSize = true;
                    if(self.nzWidth > self.nzHeight){
                        self.currentZoomLevel = self.newvaluewidth;
                    } // if
                    if(self.nzHeight > self.nzWidth){
                        self.currentZoomLevel = self.newvaluewidth;
                    } //if
                } // if
            } // if

            self.setPosition(self.currentLoc);
        },
        closeAll: function(){
            if(self.zoomWindow){self.zoomWindow.hide();}
            if(self.zoomLens){self.zoomLens.hide();}
            if(self.zoomTint){self.zoomTint.hide();}
        }, // func
        changeState: function(value){
            var self = this;
            if(value == 'enable'){self.options.zoomEnabled = true;}
            if(value == 'disable'){self.options.zoomEnabled = false;}
        } // func
    };

    $.fn.imageZoom = function( options ) {
        return this.each(function() {
            var image = Object.create( ImageZoom );
            image.init( options, this );
            $.data( this, 'imageZoom', image );
        }); // func
    }; // func

    $.fn.imageZoom.options = {
        zoomActivation: "hover",
        zoomEnabled: true,
        preloading: 1,
        zoomLevel: 1,
        scrollZoom: false,
        scrollZoomIncrement: 0.1,
        minZoomLevel: false,
        maxZoomLevel: false,
        easing: false,
        easingAmount: 12,
        lensSize: 200,
        zoomWindowWidth: 400,
        zoomWindowHeight: 400,
        zoomWindowOffetx: 0,
        zoomWindowOffety: 0,
        zoomWindowPosition: 1,
        zoomWindowBgColour: "#fff",
        lensFadeIn: false,
        lensFadeOut: false,
        debug: false,
        zoomWindowFadeIn: false,
        zoomWindowFadeOut: false,
        zoomWindowAlwaysShow: false,
        zoomTintFadeIn: false,
        zoomTintFadeOut: false,
        borderSize: 4,
        showLens: true,
        borderColour: "#888",
        lensBorderSize: 1,
        lensBorderColour: "#000",
        lensShape: "square", // or "round"
        zoomType: "window", // "window" or "lens"
        containLensZoom: false,
        lensColour: "white",
        lensOpacity: 0.4,
        lenszoom: false,
        tint: false,
        tintColour: "#333",
        tintOpacity: 0.4,
        rotatePage: false,
        gallery: false,
        galleryActiveClass: "zoomGalleryActive",
        imageCrossfade: false,
        constrainType: false,  //width or height
        constrainSize: false,  //in pixels the dimensions you want to constrain on
        loadingIcon: false,
        cursor:"default",
        responsive:true,
        onComplete: $.noop,
        onDestroy: function() {},
        onZoomedImageLoaded: function() {},
        onImageSwap: $.noop,
        onImageSwapComplete: $.noop
    }; // func

})( jQuery, window, document );